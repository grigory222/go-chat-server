package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/grigory222/go-chat-server/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	log             *slog.Logger
	storage         storage.Storage
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	jwtSecret       []byte
}

func New(
	log *slog.Logger,
	storage storage.Storage,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	jwtSecret string,
) *Service {
	return &Service{
		log:             log,
		storage:         storage,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		jwtSecret:       []byte(jwtSecret),
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *chatpb.User, err error) {
	const op = "services.auth.Login"
	log := s.log.With(slog.String("op", op), slog.String("email", email))

	dbUser, err := s.storage.UserByEmail(ctx, email)
	if err != nil {
		// Не раскрываем информацию о том, существует ли пользователь
		return "", "", nil, fmt.Errorf("%s: %w", op, models.ErrInvalidCredentials)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, fmt.Errorf("%s: %w", op, models.ErrInvalidCredentials)
	}

	accessToken, refreshToken, err = NewTokens(dbUser, s.accessTokenTTL, s.refreshTokenTTL, s.jwtSecret)
	if err != nil {
		log.Error("failed to create tokens", slog.Any("err", err))
		return "", "", nil, fmt.Errorf("%s: %w", op, err)
	}

	protoUser := &chatpb.User{
		Id:   dbUser.ID,
		Name: dbUser.Name,
	}

	return accessToken, refreshToken, protoUser, nil
}

func (s *Service) Register(ctx context.Context, name, email, password string) (*chatpb.User, error) {
	const op = "services.auth.Register"
	log := s.log.With(slog.String("op", op), slog.String("email", email))

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", slog.Any("err", err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	userID, err := s.storage.SaveUser(ctx, name, email, string(passHash))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoUser := &chatpb.User{
		Id:   userID,
		Name: name,
	}

	return protoUser, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	const op = "services.auth.RefreshToken"
	log := s.log.With(slog.String("op", op))

	// 1. Валидируем токен и получаем из него ID пользователя
	userID, err := GetUserID(refreshToken, s.jwtSecret)
	if err != nil {
		log.Warn("invalid refresh token", slog.Any("err", err))
		// Возвращаем ошибку, которую поймет gRPC слой
		return "", models.ErrInvalidCredentials
	}

	// 2. Проверяем, что пользователь с таким ID все еще существует в БД
	user, err := s.storage.UserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			log.Warn("user from token not found", slog.Int64("user_id", userID))
			return "", models.ErrInvalidCredentials
		}
		log.Error("failed to get user by id", slog.Any("err", err))
		return "", err
	}

	// 3. Генерируем новый access токен
	newAccessToken, err := newAccessToken(user, s.accessTokenTTL, s.jwtSecret)
	if err != nil {
		log.Error("failed to create new access token", slog.Any("err", err))
		return "", err
	}

	return newAccessToken, nil
}

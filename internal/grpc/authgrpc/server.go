package authgrpc

import (
	"context"
	"errors"
	"github.com/grigory222/go-chat-server/internal/storage"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"time"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"google.golang.org/grpc"
)

type serverAPI struct {
	chatpb.UnimplementedAuthServiceServer
	log             *slog.Logger
	storage         storage.Storage
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func Register(gRPC *grpc.Server, log *slog.Logger, storage storage.Storage, accessTokenTTL, refreshTokenTTL time.Duration) {
	chatpb.RegisterAuthServiceServer(gRPC, &serverAPI{
		log:             log,
		storage:         storage,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	})
}

func (s *serverAPI) Login(ctx context.Context, req *chatpb.LoginRequest) (*chatpb.LoginResponse, error) {
	const op = "authgrpc.Login"
	log := s.log.With(slog.String("op", op), slog.String("email", req.GetEmail()))

	log.Info("logging in user")

	// 1. Валидация входных данных
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// 2. Получаем пользователя из хранилища
	user, err := s.storage.UserByEmail(ctx, req.GetEmail())
	if err != nil {
		// Если пользователь не найден
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.Any("err", err))
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		// Для всех остальных ошибок
		log.Error("failed to get user", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to login")
	}

	// 3. Сравниваем хеш пароля из БД с паролем из запроса
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		log.Warn("invalid password", slog.Any("err", err))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// 4. Генерация токенов доступа (Access) и обновления (Refresh)
	// TODO: Реализовать генерацию JWT-токенов.
	// В реальном приложении здесь будет вызов сервиса, который создает
	// подписанный JWT-токен, содержащий userID, email и время жизни.
	accessToken := "mock_jwt_access_token_for_user_" + user.Name
	refreshToken := "mock_jwt_refresh_token_for_user_" + user.Name

	log.Info("user logged in successfully", slog.Int64("user_id", user.ID))

	// 5. Возвращаем ответ с токенами и информацией о пользователе
	return &chatpb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &chatpb.User{
			Id:   user.ID,
			Name: user.Name,
		},
	}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *chatpb.RegisterRequest) (*chatpb.RegisterResponse, error) {
	const op = "authgrpc.Register"
	log := s.log.With(slog.String("op", op), slog.String("email", req.GetEmail()))

	log.Info("registering user")

	if req.GetEmail() == "" || req.GetPassword() == "" || req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name, email and password are required")
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	userID, err := s.storage.SaveUser(ctx, req.GetName(), req.GetEmail(), string(passHash))
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", slog.Any("err", err))
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
		log.Error("failed to save user", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	log.Info("user registered successfully", slog.Int64("user_id", userID))

	return &chatpb.RegisterResponse{
		User: &chatpb.User{
			Id:   userID,
			Name: req.GetName(),
		},
	}, nil
}

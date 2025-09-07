package authgrpc

import (
	"context"
	"errors"
	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
)

type AuthService interface {
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *chatpb.User, err error)
	Register(ctx context.Context, name, email, password string) (user *chatpb.User, err error)
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
}

type serverAPI struct {
	chatpb.UnimplementedAuthServiceServer
	log  *slog.Logger
	auth AuthService
}

func Register(gRPC *grpc.Server, log *slog.Logger, auth AuthService) {
	chatpb.RegisterAuthServiceServer(gRPC, &serverAPI{
		log:  log,
		auth: auth,
	})
}

func (s *serverAPI) Login(ctx context.Context, req *chatpb.LoginRequest) (*chatpb.LoginResponse, error) {
	const op = "authgrpc.Login"
	log := s.log.With(slog.String("op", op), slog.String("email", req.GetEmail()))

	log.Info("logging in user")

	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	accessToken, refreshToken, user, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		// Сервис может вернуть ошибку, что пользователь не найден.
		// Мы преобразуем ее в gRPC-статус Unauthenticated.
		if errors.Is(err, models.ErrUserNotFound) {
			log.Warn("user not found or invalid credentials")
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		log.Error("failed to login user", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to login")
	}

	log.Info("user logged in successfully", slog.Int64("user_id", user.Id))

	return &chatpb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *chatpb.RegisterRequest) (*chatpb.RegisterResponse, error) {
	const op = "authgrpc.Register"
	log := s.log.With(slog.String("op", op), slog.String("email", req.GetEmail()))

	log.Info("registering user")

	if req.GetEmail() == "" || req.GetPassword() == "" || req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name, email and password are required")
	}

	user, err := s.auth.Register(ctx, req.GetName(), req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, models.ErrUserExists) {
			log.Warn("user already exists")
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
		log.Error("failed to register user", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	log.Info("user registered successfully", slog.Int64("user_id", user.Id))

	return &chatpb.RegisterResponse{
		User: user,
	}, nil
}

func (s *serverAPI) RefreshToken(ctx context.Context, req *chatpb.RefreshTokenRequest) (*chatpb.RefreshTokenResponse, error) {
	const op = "authgrpc.RefreshToken"
	log := s.log.With(slog.String("op", op))

	if req.GetRefreshToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	accessToken, err := s.auth.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		// Сервис возвращает ошибку, если токен невалиден или пользователь не найден
		if errors.Is(err, models.ErrInvalidCredentials) {
			log.Warn("invalid refresh token provided")
			return nil, status.Error(codes.Unauthenticated, "invalid or expired refresh token")
		}
		log.Error("failed to refresh token", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	log.Info("token refreshed successfully")

	return &chatpb.RefreshTokenResponse{
		AccessToken: accessToken,
	}, nil
}

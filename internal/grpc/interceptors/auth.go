package interceptors

import (
	"context"
	"log/slog"
	"strings"

	"github.com/grigory222/go-chat-server/internal/services/auth" // Нам нужен доступ к логике JWT
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// userCtxKey - это ключ для хранения ID пользователя в контексте.
type userCtxKey string

const UserIDKey = userCtxKey("userID")

// NewAuthInterceptor создает новый gRPC перехватчик для аутентификации.
func NewAuthInterceptor(log *slog.Logger, jwtSecret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		// Определяем, какие методы не требуют аутентификации
		publicMethods := map[string]bool{
			"/chat.AuthService/Login":        true,
			"/chat.AuthService/Register":     true,
			"/chat.AuthService/RefreshToken": true,
		}

		// Если вызываемый метод публичный, просто пропускаем проверку
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Получаем токен из метаданных (заголовков) запроса
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
		}

		authHeader, ok := md["authorization"]
		if !ok || len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
		}

		// Токен передается в формате "Bearer <token>"
		header := authHeader[0]
		if !strings.HasPrefix(header, "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}
		token := strings.TrimPrefix(header, "Bearer ")

		// Валидируем токен
		userID, err := auth.GetUserID(token, []byte(jwtSecret))
		if err != nil {
			log.Warn("failed to verify token", slog.Any("err", err))
			return nil, status.Error(codes.Unauthenticated, "invalid access token")
		}

		log.Debug("user authenticated", slog.Int64("user_id", userID))

		ctx = context.WithValue(ctx, UserIDKey, userID)

		return handler(ctx, req)
	}
}

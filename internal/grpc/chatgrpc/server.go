package chatgrpc

import (
	"context"
	"errors"
	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"log/slog"

	"github.com/grigory222/go-chat-server/internal/grpc/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChatService - интерфейс, который определяет потребитель (хендлер).
// Он полностью описывает, что нам нужно от сервисного слоя.
type ChatService interface {
	CreateChat(ctx context.Context, name string, userID int64) (*chatpb.Chat, error)
	GetHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*chatpb.Message, error)
	JoinChat(stream chatpb.ChatService_JoinChatServer) error
}

type serverAPI struct {
	chatpb.UnimplementedChatServiceServer
	log  *slog.Logger
	chat ChatService
}

func Register(gRPC *grpc.Server, log *slog.Logger, chat ChatService) {
	chatpb.RegisterChatServiceServer(gRPC, &serverAPI{
		log:  log,
		chat: chat,
	})
}

func (s *serverAPI) CreateChat(ctx context.Context, req *chatpb.CreateChatRequest) (*chatpb.CreateChatResponse, error) {
	const op = "grpc.chat.CreateChat"
	log := s.log.With(slog.String("op", op))

	// 1. Получаем userID из контекста (который добавил interceptor)
	userID, ok := ctx.Value(interceptors.UserIDKey).(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing user context")
	}

	// 2. Валидация
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	log.Info("creating chat", slog.String("name", req.Name), slog.Int64("user_id", userID))

	// 3. Делегируем вызов сервису
	// TODO: type of chat: private/public
	chatProto, err := s.chat.CreateChat(ctx, req.GetName(), userID)
	if err != nil {
		log.Error("failed to create chat", slog.Any("err", err))
		return nil, status.Error(codes.Internal, "failed to create chat")
	}

	return &chatpb.CreateChatResponse{Chat: chatProto}, nil
}

func (s *serverAPI) GetHistory(ctx context.Context, req *chatpb.GetHistoryRequest) (*chatpb.GetHistoryResponse, error) {
	const op = "grpc.chat.GetHistory"
	log := s.log.With(slog.String("op", op))

	// 1. Валидация
	if req.GetChatId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "chat_id is required")
	}

	limit := req.GetLimit()
	if limit == 0 || limit > 100 {
		limit = 50 // Default/max limit
	}

	log.Info("getting chat history", slog.Int64("chat_id", req.GetChatId()))

	// 2. Делегируем вызов сервису
	messages, err := s.chat.GetHistory(ctx, req.GetChatId(), uint64(limit), uint64(req.GetOffset()))
	if err != nil {
		log.Error("failed to get history", slog.Any("err", err))
		if errors.Is(err, models.ErrAccessDenied) {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
		if errors.Is(err, models.ErrChatNotFound) {
			return nil, status.Error(codes.NotFound, "chat not found")
		}
		return nil, status.Error(codes.Internal, "failed to get history")
	}

	return &chatpb.GetHistoryResponse{Messages: messages}, nil
}

// JoinChat для стриминга просто проксирует вызов в сервис.
// Вся сложная логика стрима инкапсулирована в сервисе.
func (s *serverAPI) JoinChat(stream chatpb.ChatService_JoinChatServer) error {
	const op = "grpc.chat.JoinChat"
	log := s.log.With(slog.String("op", op))
	log.Info("handling JoinChat stream")

	return s.chat.JoinChat(stream)
}

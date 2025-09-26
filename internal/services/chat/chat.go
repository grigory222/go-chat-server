package chat

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/grigory222/go-chat-server/internal/grpc/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grigory222/go-chat-server/internal/storage"
)

type Service struct {
	log     *slog.Logger
	storage storage.Storage
	hub     *Hub
}

func New(log *slog.Logger, storage storage.Storage, hub *Hub) *Service {
	return &Service{log: log, storage: storage, hub: hub}
}

func (s *Service) CreateChat(ctx context.Context, name string, userID int64) (*chatpb.Chat, error) {
	const op = "services.chat.CreateChat"

	chatID, err := s.storage.CreateChat(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := s.storage.AddUserToChat(ctx, chatID, userID); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &chatpb.Chat{Id: chatID, Name: name, Type: "public"}, nil
}

func (s *Service) GetHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*chatpb.Message, error) {
	const op = "services.chat.GetHistory"
	log := s.log.With(slog.String("op", op), slog.Int64("chat_id", chatID))

	// 0. получаем userID из контекста (interceptor уже положил его туда)
	userID, ok := ctx.Value(interceptors.UserIDKey).(int64)
	if !ok {
		log.Warn("missing user id in context")
		return nil, models.ErrInvalidCredentials // или другая подходящая доменная ошибка
	}

	// 1. проверяем членство пользователя в чате
	inChat, err := s.storage.IsUserInChat(ctx, userID, chatID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !inChat {
		log.Warn("access denied: user is not member of chat", slog.Int64("user_id", userID))
		return nil, models.ErrAccessDenied
	}

	messages, err := s.storage.GetChatHistory(ctx, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	protoMessages := make([]*chatpb.Message, len(messages))
	for i, msg := range messages {
		protoMessages[i] = &chatpb.Message{
			Id:        msg.ID,
			ChatId:    msg.ChatID,
			UserId:    msg.UserID,
			UserName:  msg.UserName,
			Text:      msg.Text,
			CreatedAt: msg.CreatedAt.Unix(),
		}
	}

	return protoMessages, nil
}

func (s *Service) JoinChat(stream chatpb.ChatService_JoinChatServer) error {
	const op = "services.chat.JoinChat"
	log := s.log.With(slog.String("op", op))

	userID, ok := stream.Context().Value(interceptors.UserIDKey).(int64)
	if !ok {
		return status.Error(codes.Internal, "failed to get user id")
	}

	initialReq, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive initial request: %v", err)
	}
	chatID := initialReq.GetChatId()

	log.Info("user connecting", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID))

	messageCh, doneCh := s.hub.Register(chatID, userID, stream)
	defer s.hub.Unregister(chatID, userID)

	// Горутина на отправку сообщений клиенту
	go func() {
		for {
			select {
			case msg := <-messageCh:
				if err := stream.Send(msg); err != nil {
					log.Error("failed to send message to client", slog.Any("err", err))
				}
			case <-doneCh:
				return
			}
		}
	}()

	// Цикл на прием сообщений от клиента
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Info("client disconnected", slog.Int64("user_id", userID))
			return nil
		}
		if err != nil {
			log.Error("stream error", slog.Any("err", err))
			return status.Errorf(codes.Unknown, "stream error: %v", err)
		}

		savedMsg, err := s.storage.SaveMessage(stream.Context(), chatID, userID, req.GetText())
		if err != nil {
			log.Error("failed to save message", slog.Any("err", err))
			continue
		}

		protoMsg := &chatpb.Message{
			Id:        savedMsg.ID,
			ChatId:    savedMsg.ChatID,
			UserId:    savedMsg.UserID,
			UserName:  savedMsg.UserName,
			Text:      savedMsg.Text,
			CreatedAt: savedMsg.CreatedAt.Unix(),
		}

		s.hub.Broadcast(protoMsg, userID)
	}
}

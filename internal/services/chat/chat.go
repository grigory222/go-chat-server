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
	log       *slog.Logger
	storage   storage.Storage
	publisher *Publisher
}

func New(log *slog.Logger, storage storage.Storage, publisher *Publisher) *Service {
	return &Service{log: log, storage: storage, publisher: publisher}
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

	userID, ok := ctx.Value(interceptors.UserIDKey).(int64)
	if !ok {
		log.Warn("missing user id in context")
		return nil, models.ErrInvalidCredentials
	}

	// Проверяем что пользователь состоит в чате
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

	subscriber := newChatSubscriber(userID, stream, log)
	s.publisher.Register(chatID, subscriber)
	defer s.publisher.Unregister(chatID, userID)

	// Читаем сообщения от клиента
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

		s.publisher.Broadcast(protoMsg, userID)
	}
}

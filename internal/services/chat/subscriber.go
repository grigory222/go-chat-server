package chat

import (
	"log/slog"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
)

// Subscriber получает уведомления о новых сообщениях в чате
type Subscriber interface {
	// Notify отправляет сообщение подписчику
	Notify(msg *chatpb.Message)

	// ID возвращает user ID подписчика
	ID() int64

	// Close закрывает подписчика
	Close()
}

// chatSubscriber отправляет сообщения клиенту через gRPC stream
type chatSubscriber struct {
	userID    int64
	stream    chatpb.ChatService_JoinChatServer
	messageCh chan *chatpb.Message
	doneCh    chan struct{}
	log       *slog.Logger
}

// newChatSubscriber создает подписчика и запускает горутину для отправки
func newChatSubscriber(userID int64, stream chatpb.ChatService_JoinChatServer, log *slog.Logger) *chatSubscriber {
	sub := &chatSubscriber{
		userID:    userID,
		stream:    stream,
		messageCh: make(chan *chatpb.Message, 10),
		doneCh:    make(chan struct{}),
		log:       log,
	}

	// Запускаем writer-горутину для отправки сообщений клиенту
	go sub.writerLoop()

	return sub
}

// writerLoop читает из канала и отправляет сообщения клиенту
func (s *chatSubscriber) writerLoop() {
	for {
		select {
		case msg := <-s.messageCh:
			if err := s.stream.Send(msg); err != nil {
				s.log.Error("failed to send message to client",
					slog.Int64("user_id", s.userID),
					slog.Any("err", err))
				// Продолжаем работу, основной цикл сам обнаружит disconnect
			}
		case <-s.doneCh:
			s.log.Debug("writer loop terminated", slog.Int64("user_id", s.userID))
			return
		}
	}
}

// Notify кладет сообщение в канал подписчика
func (s *chatSubscriber) Notify(msg *chatpb.Message) {
	select {
	case s.messageCh <- msg:
		// Сообщение успешно поставлено в очередь на отправку
	default:
		// Канал переполнен, дропаем сообщение чтобы не блокировать рассылку
		s.log.Warn("subscriber message channel is full, dropping message",
			slog.Int64("user_id", s.userID),
			slog.Int64("chat_id", msg.GetChatId()),
		)
	}
}

// ID возвращает user ID подписчика
func (s *chatSubscriber) ID() int64 {
	return s.userID
}

// Close останавливает writer goroutine
func (s *chatSubscriber) Close() {
	close(s.doneCh)
}

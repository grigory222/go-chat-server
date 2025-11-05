package chat

import (
	"log/slog"
	"sync"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
)

// Publisher управляет подписчиками и рассылает им сообщения
type Publisher struct {
	log *slog.Logger
	mu  sync.RWMutex

	// subscribers хранит подписчиков по chatID и userID
	subscribers map[int64]map[int64]Subscriber
}

// NewPublisher создает новый Publisher
func NewPublisher(log *slog.Logger) *Publisher {
	return &Publisher{
		log:         log,
		subscribers: make(map[int64]map[int64]Subscriber),
	}
}

// Register добавляет подписчика в чат
func (p *Publisher) Register(chatID int64, subscriber Subscriber) {
	p.mu.Lock()
	defer p.mu.Unlock()

	userID := subscriber.ID()

	if _, ok := p.subscribers[chatID]; !ok {
		p.subscribers[chatID] = make(map[int64]Subscriber)
	}

	p.subscribers[chatID][userID] = subscriber

	p.log.Info("subscriber registered in publisher", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID))
}

// Unregister удаляет подписчика из чата
func (p *Publisher) Unregister(chatID, userID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if chatSubscribers, ok := p.subscribers[chatID]; ok {
		if subscriber, ok := chatSubscribers[userID]; ok {
			subscriber.Close()
			delete(p.subscribers[chatID], userID)
		}
		// Если чат пустой, удаляем его из карты
		if len(p.subscribers[chatID]) == 0 {
			delete(p.subscribers, chatID)
		}
	}

	p.log.Info("subscriber unregistered from publisher", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID))
}

// Broadcast рассылает сообщение всем подписчикам чата кроме отправителя
func (p *Publisher) Broadcast(msg *chatpb.Message, senderID int64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	chatID := msg.GetChatId()

	if chatSubscribers, ok := p.subscribers[chatID]; ok {
		p.log.Debug("broadcasting message",
			slog.Int64("chat_id", chatID),
			slog.Int64("sender_id", senderID),
			slog.Int("subscribers_in_chat", len(chatSubscribers)),
		)

		for userID, subscriber := range chatSubscribers {
			if userID == senderID {
				continue
			}

			subscriber.Notify(msg)
		}
	}
}

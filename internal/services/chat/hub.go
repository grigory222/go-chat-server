package chat

import (
	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"log/slog"
	"sync"
)

// Внутренняя структура Хаба, представляющая одно клиентское подключение.
// Она хранит сам gRPC-поток и каналы для общения с горутиной-писателем.
type clientStream struct {
	stream    chatpb.ChatService_JoinChatServer // Поток для отправки сообщений клиенту
	messageCh chan *chatpb.Message              // Канал, куда Хаб кладет сообщения для отправки
	doneCh    chan struct{}                     // Канал для сигнала о завершении работы
}

// Hub - центральный диспетчер, который управляет всеми активными подключениями.
type Hub struct {
	log *slog.Logger
	mu  sync.RWMutex

	// Основное хранилище. Структура: map[ID чата] -> map[ID пользователя] -> *clientStream
	clients map[int64]map[int64]*clientStream
}

// NewHub создает и возвращает новый экземпляр Хаба.
func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		log:     log,
		clients: make(map[int64]map[int64]*clientStream),
	}
}

// Register регистрирует новый клиентский поток в Хабе.
// Эта функция создает каналы для клиента и возвращает их сервисному слою.
func (h *Hub) Register(chatID, userID int64, stream chatpb.ChatService_JoinChatServer) (chan *chatpb.Message, chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Если для этого чата еще нет карты клиентов, создаем ее
	if _, ok := h.clients[chatID]; !ok {
		h.clients[chatID] = make(map[int64]*clientStream)
	}

	// Создаем новую структуру для клиента
	client := &clientStream{
		stream:    stream,
		messageCh: make(chan *chatpb.Message, 10),
		doneCh:    make(chan struct{}),
	}

	// Сохраняем клиента в карте
	h.clients[chatID][userID] = client

	h.log.Info("client registered in hub", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID))

	// Возвращаем каналы, чтобы горутины в сервисном слое могли с ними работать
	return client.messageCh, client.doneCh
}

// Unregister удаляет клиентский поток из Хаба.
func (h *Hub) Unregister(chatID, userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Проверяем, существует ли вообще запись о таком чате
	if chatClients, ok := h.clients[chatID]; ok {
		// Проверяем, существует ли запись о таком клиенте
		if client, ok := chatClients[userID]; ok {
			// Закрываем канал `doneCh`, чтобы горутина-писатель завершила свою работу
			close(client.doneCh)
			// Удаляем клиента из карты чата
			delete(h.clients[chatID], userID)
		}
		// Если в чате больше не осталось клиентов, удаляем и сам чат из карты
		if len(h.clients[chatID]) == 0 {
			delete(h.clients, chatID)
		}
	}

	h.log.Info("client unregistered from hub", slog.Int64("user_id", userID), slog.Int64("chat_id", chatID))
}

// Broadcast рассылает сообщение всем клиентам в указанном чате, кроме самого отправителя.
func (h *Hub) Broadcast(msg *chatpb.Message, senderID int64) {
	h.mu.RLock() // Блокировка только на чтение, чтобы не мешать другим рассылкам
	defer h.mu.RUnlock()

	chatID := msg.GetChatId()

	// Проверяем, есть ли активные клиенты в этом чате
	if chatClients, ok := h.clients[chatID]; ok {
		h.log.Debug("broadcasting message",
			slog.Int64("chat_id", chatID),
			slog.Int64("sender_id", senderID),
			slog.Int("clients_in_chat", len(chatClients)),
		)

		for userID, client := range chatClients {
			if userID == senderID {
				continue
			}

			// Неблокирующая отправка в канал клиента
			select {
			case client.messageCh <- msg:
				// Сообщение успешно поставлено в очередь на отправку
			default:
				// Канал клиента переполнен (он не успевает получать сообщения)
				// Чтобы весь код не завис, придётся пожертвовать доставкой сообщения одному медленному клиенту
				h.log.Warn("client message channel is full, dropping message",
					slog.Int64("user_id", userID),
					slog.Int64("chat_id", chatID),
				)
			}
		}
	}
}

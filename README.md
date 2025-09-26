## go-chat-server

Простой gRPC чат: стрим сообщений, JWT авторизация, PostgreSQL.

### Демонстрация
`.docs/media/demo.gif`

### Технологии
Go • gRPC • Protobuf • PostgreSQL • JWT • bcrypt • slog

### Функциональность
* Регистрация / логин / refresh токен
* Создание чата
* История сообщений
* Стрим: подключение к чату и получение новых сообщений
* Авторизация через interceptor (все приватные методы проверяются автоматически)
* Разделение: transport / services / storage
* Hub для рассылки без блокировок
* Тесты сервиса чата (ошибки + стрим)


### Запуск сервера
```bash
docker compose up -d
go run ./cmd/server
```


### API
Auth: Login, Register, RefreshToken
Chat: CreateChat, GetHistory, JoinChat(stream)

### Hub
Держит активные подключения по chatID. Сообщения рассылает всем кроме отправителя. Если канал клиента переполнен — сообщение пропускается.

### Тестирование
`internal/services/chat/chat_test.go` — базовое покрытие потоков и ошибок.

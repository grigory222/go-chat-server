## Go streaming chat 

<p>
	<a href="https://github.com/grigory222/go-chat-server/actions/workflows/ci.yml">
		<img src="https://github.com/grigory222/go-chat-server/actions/workflows/ci.yml/badge.svg" alt="CI" />
	</a>
	<a href="https://github.com/grigory222/go-chat-server">
		<img src="https://img.shields.io/badge/go-1.24+-00ADD8?logo=go" alt="Go Version" />
	</a>
	<!-- Coverage badge placeholder. After enabling publishing (см. ниже), файл coverage.svg станет доступен. -->
	<img src="https://raw.githubusercontent.com/grigory222/go-chat-server/gh-assets/coverage.svg" alt="Coverage" />
</p>

> Примечание: текущий workflow генерирует coverage.svg как артефакт. Чтобы ссылка на бейдж работала постоянно, опубликуйте файл (см. раздел "Coverage badge" ниже).

### Демонстрация

![Demo](.docs/media/demo.gif)

### Назначение
Сервис предоставляет API для обмена сообщениями в чатах в режиме реального времени (gRPC streaming) с поддержкой аутентификации пользователей. Архитектура ориентирована на расширяемость (новые типы событий, приватные чаты, масштабирование Hub).

### Связанные репозитории
В этом репозитории представлена серверная часть. Протокол и клиент находятся отдельно:
* Протоколы (protobuf): https://github.com/grigory222/go-chat-proto
* Клиент: https://github.com/grigory222/go-chat-client

### Технологии
Go | gRPC | Protobuf | PostgreSQL | pgx | JWT | bcrypt | slog | cleanenv

### Основные возможности
* JWT: access / refresh токены
* Создание чатов
* Хранение истории сообщений
* Двунаправленный gRPC stream для доставки новых сообщений
* Интерсептор авторизации
* Изолированный слой хранения
* Hub: неблокирующая широковещательная рассылка
* Юнит-тесты сервисного слоя


### Быстрый запуск
```bash
docker compose up -d
go run ./cmd/server
```

### API
Ниже перечислены методы и их назначение. Подробные сигнатуры и типы см. в protobuf (`go-chat-proto`).

AuthService
* Register – создание нового пользователя
* Login – проверка учётных данных и выдача пары токенов (access + refresh)
* RefreshToken – получение нового access токена по действующему refresh

ChatService
* CreateChat – создаёт чат и автоматически добавляет инициатора
* GetHistory – постраничная выборка истории сообщений (проверяется членство в чате)
* JoinChat – устанавливает streaming-сессию для отправки и получения новых сообщений в реальном времени

JoinChat (процесс):
1. Первое входящее сообщение клиента содержит `chat_id` (регистрация подключения).
2. Далее клиент отправляет текстовые сообщения.
3. Сервер сохраняет сообщение и отправляет его остальным участникам чата (кроме отправителя).

Сообщение (`Message`): `id, chat_id, user_id, user_name, text, created_at (unix)`.

### Coverage badge (как сделать постоянным)
По умолчанию бейдж доступен только как артефакт CI. Варианты публикации:
1. Ветка gh-assets: workflow коммитит `coverage.svg` туда после тестов.
2. GitHub Pages: деплойте артефакт в `gh-pages` и укажите URL.
3. Codecov / Coveralls: подключите внешний сервис и замените тег.

Пример шага (добавьте в конец job) для ветки gh-assets (требует `permissions.contents: write`):
```
			- name: Publish coverage badge to gh-assets branch
				if: github.ref == 'refs/heads/main'
				run: |
					git fetch origin gh-assets || true
					git checkout -B gh-assets
					cp .github/badges/coverage.svg coverage.svg
					git add coverage.svg
					git -c user.name=ci -c user.email=ci@example.com commit -m "update coverage badge" || true
					git push origin gh-assets
```

После этого тег в начале README начнёт отображать актуальное покрытие.


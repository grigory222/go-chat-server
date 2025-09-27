package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/grigory222/go-chat-server/internal/config"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func New(ctx context.Context, cfg config.Postgres, log *slog.Logger) (*Storage, error) {
	const op = "storage.postgres.New"

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to parse config: %w", op, err)
	}

	// Apply sane defaults if zero values (when config loaded manually in tests without env/yaml defaults)
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 10
	}
	if cfg.MinConns < 0 {
		cfg.MinConns = 0
	}
	// Ensure MinConns does not exceed MaxConns
	if cfg.MinConns > cfg.MaxConns {
		cfg.MinConns = cfg.MaxConns
	}
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect to postgres: %w", op, err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("%s: failed to ping postgres: %w", op, err)
	}

	log.Info("connected to PostgreSQL", slog.String("db_name", cfg.DBName))

	return &Storage{pool: pool, log: log}, nil
}

func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Storage) SaveUser(ctx context.Context, name, email, passHash string) (int64, error) {
	const op = "storage.postgres.SaveUser"

	query := `INSERT INTO users (name, email, password_hash) VALUES (@name, @email, @passwordHash) RETURNING id`
	args := pgx.NamedArgs{
		"name":         name,
		"email":        email,
		"passwordHash": passHash,
	}

	var id int64
	err := s.pool.QueryRow(ctx, query, args).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		// Проверяем, является ли ошибка специфической ошибкой PostgreSQL
		if errors.As(err, &pgErr) {
			// Код '23505' в PostgreSQL означает 'unique_violation' (нарушение уникальности)
			if pgErr.Code == "23505" {
				return 0, fmt.Errorf("%s: %w", op, models.ErrUserExists)
			}
		}
		s.log.Error("failed to save user", slog.Any("err", err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) UserByEmail(ctx context.Context, email string) (*models.User, error) {
	const op = "storage.postgres.UserByEmail"

	query := `SELECT id, name, email, password_hash FROM users WHERE email = @email`
	args := pgx.NamedArgs{
		"email": email,
	}

	var user models.User
	err := s.pool.QueryRow(ctx, query, args).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, models.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (s *Storage) UserByID(ctx context.Context, id int64) (*models.User, error) {
	const op = "storage.postgres.UserByID"

	query := `SELECT id, name, email, password_hash FROM users WHERE id = @id`
	args := pgx.NamedArgs{"id": id}

	var user models.User
	err := s.pool.QueryRow(ctx, query, args).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, models.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

// CreateChat создает новый чат и возвращает его ID.
func (s *Storage) CreateChat(ctx context.Context, name string) (int64, error) {
	const op = "storage.postgres.CreateChat"

	query := `INSERT INTO chats (name, type) VALUES (@name, 'public') RETURNING id` // Пока все чаты публичные
	args := pgx.NamedArgs{"name": name}

	var id int64
	if err := s.pool.QueryRow(ctx, query, args).Scan(&id); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// AddUserToChat добавляет пользователя в чат.
func (s *Storage) AddUserToChat(ctx context.Context, chatID, userID int64) error {
	const op = "storage.postgres.AddUserToChat"

	query := `INSERT INTO chat_users (chat_id, user_id) VALUES (@chatID, @userID) ON CONFLICT DO NOTHING`
	args := pgx.NamedArgs{"chatID": chatID, "userID": userID}

	if _, err := s.pool.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// SaveMessage сохраняет новое сообщение в БД и возвращает его полную модель.
func (s *Storage) SaveMessage(ctx context.Context, chatID, userID int64, text string) (*models.Message, error) {
	const op = "storage.postgres.SaveMessage"

	// Сначала вставляем сообщение
	query := `INSERT INTO messages (chat_id, user_id, text) VALUES (@chatID, @userID, @text) 
	          RETURNING id, created_at`
	args := pgx.NamedArgs{"chatID": chatID, "userID": userID, "text": text}

	var msg models.Message
	msg.ChatID = chatID
	msg.UserID = userID
	msg.Text = text

	if err := s.pool.QueryRow(ctx, query, args).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Затем получаем имя пользователя
	userQuery := `SELECT name FROM users WHERE id = @userID`
	if err := s.pool.QueryRow(ctx, userQuery, pgx.NamedArgs{"userID": userID}).Scan(&msg.UserName); err != nil {
		return nil, fmt.Errorf("%s: failed to get user name: %w", op, err)
	}

	return &msg, nil
}

// GetChatHistory получает историю сообщений из чата с пагинацией.
func (s *Storage) GetChatHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*models.Message, error) {
	const op = "storage.postgres.GetChatHistory"

	query := `SELECT m.id, m.chat_id, m.user_id, u.name, m.text, m.created_at 
	          FROM messages m 
	          JOIN users u ON m.user_id = u.id 
	          WHERE m.chat_id = @chatID 
	          ORDER BY m.created_at DESC LIMIT @limit OFFSET @offset`
	args := pgx.NamedArgs{"chatID": chatID, "limit": limit, "offset": offset}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.UserID, &msg.UserName, &msg.Text, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return messages, nil
}

// IsUserInChat проверяет, состоит ли пользователь в чате.
func (s *Storage) IsUserInChat(ctx context.Context, userID, chatID int64) (bool, error) {
	const op = "storage.postgres.IsUserInChat"

	query := `SELECT 1 FROM chat_users WHERE user_id = @userID AND chat_id = @chatID LIMIT 1`
	args := pgx.NamedArgs{"userID": userID, "chatID": chatID}

	var tmp int
	err := s.pool.QueryRow(ctx, query, args).Scan(&tmp)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return true, nil
}

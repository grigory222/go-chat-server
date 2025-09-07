package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/grigory222/go-chat-server/internal/config"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
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

// TODO: Реализовать остальные методы для работы с чатами и сообщениями
// func (s *Storage) CreateChat(...) (...)
// func (s *Storage) GetChatHistory(...) (...)
// func (s *Storage) SaveMessage(...) (...)

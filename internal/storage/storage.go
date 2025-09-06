package storage

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// User представляет модель пользователя из БД
type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
}

type Storage interface {
	SaveUser(ctx context.Context, name, email, passHash string) (int64, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	// TODO: Добавить методы для ChatService
	// CreateChat(ctx context.Context, name string) (int64, error)
	// ... и так далее
}

package storage

import (
	"context"
	"github.com/grigory222/go-chat-server/internal/domain/models"
)

type Storage interface {
	SaveUser(ctx context.Context, name, email, passHash string) (int64, error)
	UserByEmail(ctx context.Context, email string) (*models.User, error)
	UserByID(ctx context.Context, id int64) (*models.User, error)

	// TODO: Добавить методы для ChatService
	// CreateChat(ctx context.Context, name string) (int64, error)
	// ... и так далее
}

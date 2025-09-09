package storage

import (
	"context"
	"github.com/grigory222/go-chat-server/internal/domain/models"
)

type Storage interface {
	SaveUser(ctx context.Context, name, email, passHash string) (int64, error)
	UserByEmail(ctx context.Context, email string) (*models.User, error)
	UserByID(ctx context.Context, id int64) (*models.User, error)

	CreateChat(ctx context.Context, name string) (int64, error)
	AddUserToChat(ctx context.Context, chatID, userID int64) error
	SaveMessage(ctx context.Context, chatID, userID int64, text string) (*models.Message, error)
	GetChatHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*models.Message, error)
}

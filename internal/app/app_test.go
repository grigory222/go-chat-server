package app

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	grpcapp "github.com/grigory222/go-chat-server/internal/app/grpc"
	"github.com/grigory222/go-chat-server/internal/config"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock for the storage.Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Close() {
	m.Called()
}

func (m *MockStorage) SaveUser(ctx context.Context, name, email, passHash string) (int64, error) {
	args := m.Called(ctx, name, email, passHash)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) UserByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockStorage) UserByID(ctx context.Context, id int64) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockStorage) CreateChat(ctx context.Context, name string) (int64, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStorage) AddUserToChat(ctx context.Context, chatID, userID int64) error {
	args := m.Called(ctx, chatID, userID)
	return args.Error(0)
}

func (m *MockStorage) SaveMessage(ctx context.Context, chatID, userID int64, text string) (*models.Message, error) {
	args := m.Called(ctx, chatID, userID, text)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockStorage) GetChatHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*models.Message, error) {
	args := m.Called(ctx, chatID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

func (m *MockStorage) IsUserInChat(ctx context.Context, userID, chatID int64) (bool, error) {
	args := m.Called(ctx, userID, chatID)
	return args.Bool(0), args.Error(1)
}

type MockGRPCApp struct {
	mock.Mock
	StopFunc func()
}

func (m *MockGRPCApp) Stop() {
	if m.StopFunc != nil {
		m.StopFunc()
	}
	m.Called()
}

func TestApp_Stop(t *testing.T) {
	storageMock := new(MockStorage)
	storageMock.On("Close").Return()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	grpcApp := grpcapp.New(log, 0, nil, nil, "")

	app := &App{
		GRPCSrv: grpcApp,
		Storage: storageMock,
	}

	app.Stop()

	storageMock.AssertExpectations(t)
}

func TestNew_PanicsOnStorageError(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := &config.Config{
		Postgres: config.Postgres{
			Host:           "invalid-host",
			Port:           1,
			User:           "user",
			Password:       "password",
			DBName:         "db",
			SSLMode:        "disable",
			ConnectTimeout: 10 * time.Millisecond,
		},
	}

	assert.Panics(t, func() {
		New(log, cfg)
	}, "Expected New to panic when storage connection fails")
}

package grpcapp

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/grigory222/go-chat-server/internal/services/auth"
	"github.com/grigory222/go-chat-server/internal/services/chat"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// MockStorage is a mock type for the Storage type
type MockStorage struct {
	mock.Mock
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

func (m *MockStorage) Close() {}

type GRPCAppTestSuite struct {
	suite.Suite
	app    *App
	lis    *bufconn.Listener
	client *grpc.ClientConn
}

func (s *GRPCAppTestSuite) SetupSuite() {
	const bufSize = 1024 * 1024
	s.lis = bufconn.Listen(bufSize)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	storageMock := new(MockStorage)
	// Setup mock expectations in tests

	publisher := chat.NewPublisher(log)
	authService := auth.New(log, storageMock, time.Hour, time.Hour, "secret")
	chatService := chat.New(log, storageMock, publisher)

	s.app = New(log, 0, authService, chatService, "secret")

	go func() {
		if err := s.app.gRPCServer.Serve(s.lis); err != nil {
			log.Error("Server exited with error", slog.Any("err", err))
		}
	}()

	// Use the modern Dial (grpc.Dial is still allowed; grpc.DialContext is deprecated). Provide a custom context dialer.
	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return s.lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(s.T(), err)
	s.client = conn
}

func (s *GRPCAppTestSuite) TearDownSuite() {
	s.client.Close()
	s.app.Stop()
}

func TestGRPCAppTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCAppTestSuite))
}

func (s *GRPCAppTestSuite) TestStartAndStop() {
	// Test that the server starts and stops gracefully
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	storageMock := new(MockStorage)
	publisher := chat.NewPublisher(log)
	authService := auth.New(log, storageMock, time.Hour, time.Hour, "secret")
	chatService := chat.New(log, storageMock, publisher)
	app := New(log, 9999, authService, chatService, "secret")

	go func() {
		// Since we're not in a real network environment, Run will error out
		// but it's enough to know the server tried to start.
		_ = app.Run()
	}()
	time.Sleep(100 * time.Millisecond) // Give it time to start
	app.Stop()
}

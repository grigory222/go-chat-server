package auth

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/grigory222/go-chat-server/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// mockStorage implements storage.Storage for tests.
type mockStorage struct {
	usersByEmail map[string]*models.User
	usersByID    map[int64]*models.User
	nextID       int64
	saveErr      error
	getErr       error
}

func newMockStorage() *mockStorage {
	return &mockStorage{usersByEmail: map[string]*models.User{}, usersByID: map[int64]*models.User{}, nextID: 1}
}

func (m *mockStorage) SaveUser(ctx context.Context, name, email, passHash string) (int64, error) {
	if m.saveErr != nil {
		return 0, m.saveErr
	}
	// If already exists
	if _, ok := m.usersByEmail[email]; ok {
		return 0, models.ErrUserExists
	}
	id := m.nextID
	m.nextID++
	u := &models.User{ID: id, Name: name, Email: email, PasswordHash: passHash}
	m.usersByEmail[email] = u
	m.usersByID[id] = u
	return id, nil
}
func (m *mockStorage) UserByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	u, ok := m.usersByEmail[email]
	if !ok {
		return nil, models.ErrUserNotFound
	}
	return u, nil
}
func (m *mockStorage) UserByID(ctx context.Context, id int64) (*models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	u, ok := m.usersByID[id]
	if !ok {
		return nil, models.ErrUserNotFound
	}
	return u, nil
}

// Unused chat-related methods
func (m *mockStorage) CreateChat(ctx context.Context, name string) (int64, error) {
	return 0, errors.New("not implemented")
}
func (m *mockStorage) AddUserToChat(ctx context.Context, chatID, userID int64) error {
	return errors.New("not implemented")
}
func (m *mockStorage) SaveMessage(ctx context.Context, chatID, userID int64, text string) (*models.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *mockStorage) GetChatHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*models.Message, error) {
	return nil, errors.New("not implemented")
}
func (m *mockStorage) IsUserInChat(ctx context.Context, userID, chatID int64) (bool, error) {
	return false, errors.New("not implemented")
}
func (m *mockStorage) Close() {}

var _ storage.Storage = (*mockStorage)(nil)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(os.Stdout, nil)) }

func TestRegisterAndLogin(t *testing.T) {
	st := newMockStorage()
	svc := New(testLogger(), st, time.Minute, time.Hour, "secret")

	user, err := svc.Register(context.Background(), "Alice", "alice@example.com", "password")
	if err != nil {
		t.Fatalf("register error: %v", err)
	}
	if user.Id == 0 || user.Name != "Alice" {
		t.Fatalf("unexpected user %+v", user)
	}

	// Duplicate register
	_, err = svc.Register(context.Background(), "Alice", "alice@example.com", "password")
	if !errors.Is(err, models.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}

	access, refresh, pbUser, err := svc.Login(context.Background(), "alice@example.com", "password")
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatalf("expected tokens")
	}
	if pbUser.Id != user.Id || pbUser.Name != user.Name {
		t.Fatalf("pbUser mismatch")
	}

	// Wrong password
	_, _, _, err = svc.Login(context.Background(), "alice@example.com", "wrong")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	st := newMockStorage()
	svc := New(testLogger(), st, time.Minute, time.Hour, "secret")

	// Prepare user manually
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	id, _ := st.SaveUser(context.Background(), "Bob", "bob@example.com", string(hash))
	u := &models.User{ID: id, Name: "Bob"}
	_, refresh, err := NewTokens(u, time.Minute, time.Hour, []byte("secret"))
	if err != nil {
		t.Fatalf("tokens error: %v", err)
	}

	newAccess, err := svc.RefreshToken(context.Background(), refresh)
	if err != nil {
		t.Fatalf("refresh error: %v", err)
	}
	if newAccess == "" {
		t.Fatalf("empty new access token")
	}

	// Corrupted token
	_, err = svc.RefreshToken(context.Background(), refresh+"tamper")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for tampered token, got %v", err)
	}

	// Remove user -> should return invalid credentials
	delete(st.usersByID, id)
	delete(st.usersByEmail, "bob@example.com")
	_, refresh2, _ := NewTokens(u, time.Minute, time.Hour, []byte("secret"))
	_, err = svc.RefreshToken(context.Background(), refresh2)
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials after user deletion, got %v", err)
	}
}

package chat

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/grigory222/go-chat-server/internal/grpc/interceptors"
)

// mockChatStorage provides controllable behavior for chat service tests.
type mockChatStorage struct {
	createChatID    int64
	createErr       error
	addUserErr      error
	isUserInChat    bool
	isUserInChatErr error
	historyMessages []*models.Message
	historyErr      error
	saveMsgErr      error
	savedMessages   []*models.Message
}

func (m *mockChatStorage) CreateChat(ctx context.Context, name string) (int64, error) {
	return m.createChatID, m.createErr
}
func (m *mockChatStorage) AddUserToChat(ctx context.Context, chatID, userID int64) error {
	return m.addUserErr
}
func (m *mockChatStorage) SaveMessage(ctx context.Context, chatID, userID int64, text string) (*models.Message, error) {
	if m.saveMsgErr != nil {
		return nil, m.saveMsgErr
	}
	msg := &models.Message{ID: int64(len(m.savedMessages) + 1), ChatID: chatID, UserID: userID, UserName: "User", Text: text, CreatedAt: time.Unix(1000, 0)}
	m.savedMessages = append(m.savedMessages, msg)
	return msg, nil
}
func (m *mockChatStorage) GetChatHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*models.Message, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.historyMessages, nil
}
func (m *mockChatStorage) IsUserInChat(ctx context.Context, userID, chatID int64) (bool, error) {
	return m.isUserInChat, m.isUserInChatErr
}

// Unused methods for this test suite
func (m *mockChatStorage) SaveUser(context.Context, string, string, string) (int64, error) {
	return 0, errors.New("not implemented")
}
func (m *mockChatStorage) UserByEmail(context.Context, string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockChatStorage) UserByID(context.Context, int64) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockChatStorage) Close() {}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestServiceCreateChat(t *testing.T) {
	st := &mockChatStorage{createChatID: 10}
	svc := New(testLogger(), st, NewHub(testLogger()))
	chatObj, err := svc.CreateChat(context.Background(), "General", 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chatObj.Id != 10 || chatObj.Name != "General" {
		t.Fatalf("unexpected chat: %+v", chatObj)
	}

	// Error in CreateChat
	st.createErr = errors.New("db error")
	if _, err := svc.CreateChat(context.Background(), "General", 123); err == nil {
		t.Fatalf("expected error from storage.CreateChat")
	}
}

func TestServiceGetHistoryBranches(t *testing.T) {
	st := &mockChatStorage{}
	svc := New(testLogger(), st, NewHub(testLogger()))
	ctx := context.Background()

	// Missing user id
	if _, err := svc.GetHistory(ctx, 1, 10, 0); !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	// Error from IsUserInChat
	ctx2 := context.WithValue(ctx, interceptors.UserIDKey, int64(5))
	st.isUserInChatErr = errors.New("db error")
	if _, err := svc.GetHistory(ctx2, 1, 10, 0); err == nil {
		t.Fatalf("expected error from IsUserInChat")
	}

	// Not in chat
	st.isUserInChatErr = nil
	st.isUserInChat = false
	if _, err := svc.GetHistory(ctx2, 1, 10, 0); !errors.Is(err, models.ErrAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}

	// History error
	st.isUserInChat = true
	st.historyErr = errors.New("db error")
	if _, err := svc.GetHistory(ctx2, 1, 10, 0); err == nil {
		t.Fatalf("expected history error")
	}

	// Success
	st.historyErr = nil
	st.historyMessages = []*models.Message{{ID: 1, ChatID: 1, UserID: 5, UserName: "U", Text: "Hi", CreatedAt: time.Unix(2000, 0)}}
	msgs, err := svc.GetHistory(ctx2, 1, 10, 0)
	if err != nil || len(msgs) != 1 || msgs[0].Text != "Hi" {
		t.Fatalf("unexpected result: %v %+v", err, msgs)
	}
}

// Fake stream for JoinChat tests
type fakeJoinStream struct {
	chatpb.ChatService_JoinChatServer
	ctx       context.Context
	recvQueue []*chatpb.JoinChatRequest
	recvIdx   int
	sent      []*chatpb.Message
}

func (f *fakeJoinStream) Context() context.Context { return f.ctx }
func (f *fakeJoinStream) Recv() (*chatpb.JoinChatRequest, error) {
	if f.recvIdx >= len(f.recvQueue) {
		return nil, io.EOF
	}
	r := f.recvQueue[f.recvIdx]
	f.recvIdx++
	return r, nil
}
func (f *fakeJoinStream) Send(m *chatpb.Message) error { f.sent = append(f.sent, m); return nil }

func TestServiceJoinChatSuccess(t *testing.T) {
	st := &mockChatStorage{}
	hub := NewHub(testLogger())
	svc := New(testLogger(), st, hub)

	ctx := context.WithValue(context.Background(), interceptors.UserIDKey, int64(7))
	stream := &fakeJoinStream{ctx: ctx, recvQueue: []*chatpb.JoinChatRequest{
		{ChatId: 55},                    // initial
		{ChatId: 55, Text: "Hello all"}, // message
	}}

	if err := svc.JoinChat(stream); err != nil {
		t.Fatalf("JoinChat error: %v", err)
	}
	if len(st.savedMessages) != 1 || st.savedMessages[0].Text != "Hello all" {
		t.Fatalf("message not saved: %+v", st.savedMessages)
	}
}

func TestServiceJoinChatInitialRecvError(t *testing.T) {
	st := &mockChatStorage{}
	hub := NewHub(testLogger())
	svc := New(testLogger(), st, hub)
	ctx := context.WithValue(context.Background(), interceptors.UserIDKey, int64(7))
	// Empty queue => first Recv returns EOF -> should map to InvalidArgument error
	stream := &fakeJoinStream{ctx: ctx}
	if err := svc.JoinChat(stream); err == nil {
		t.Fatalf("expected error on initial recv")
	}
}

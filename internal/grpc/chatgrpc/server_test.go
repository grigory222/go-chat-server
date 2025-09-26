package chatgrpc

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"github.com/grigory222/go-chat-server/internal/grpc/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeChatService struct {
	createResp *chatpb.Chat
	createErr  error
	histMsgs   []*chatpb.Message
	histErr    error
	joinErr    error
}

func (f *fakeChatService) CreateChat(ctx context.Context, name string, userID int64) (*chatpb.Chat, error) {
	return f.createResp, f.createErr
}
func (f *fakeChatService) GetHistory(ctx context.Context, chatID int64, limit, offset uint64) ([]*chatpb.Message, error) {
	return f.histMsgs, f.histErr
}
func (f *fakeChatService) JoinChat(stream chatpb.ChatService_JoinChatServer) error { return f.joinErr }

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(os.Stdout, nil)) }

func TestCreateChatHandler(t *testing.T) {
	api := &serverAPI{chat: &fakeChatService{createResp: &chatpb.Chat{Id: 1, Name: "Gen"}}, log: logger()}
	// Missing user context
	if _, err := api.CreateChat(context.Background(), &chatpb.CreateChatRequest{Name: "Gen"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
	// Invalid argument
	ctx := context.WithValue(context.Background(), interceptors.UserIDKey, int64(5))
	if _, err := api.CreateChat(ctx, &chatpb.CreateChatRequest{Name: ""}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument")
	}
	// Internal error
	api.chat.(*fakeChatService).createErr = errors.New("db")
	if _, err := api.CreateChat(ctx, &chatpb.CreateChatRequest{Name: "Gen"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal")
	}
	// Success
	api.chat.(*fakeChatService).createErr = nil
	if resp, err := api.CreateChat(ctx, &chatpb.CreateChatRequest{Name: "Gen"}); err != nil || resp.Chat.Name != "Gen" {
		t.Fatalf("unexpected: %v %v", err, resp)
	}
}

func TestGetHistoryHandler(t *testing.T) {
	api := &serverAPI{chat: &fakeChatService{}, log: logger()}
	// Invalid argument (chat id)
	if _, err := api.GetHistory(context.Background(), &chatpb.GetHistoryRequest{ChatId: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument")
	}
	ctx := context.WithValue(context.Background(), interceptors.UserIDKey, int64(5))
	// Access denied
	api.chat.(*fakeChatService).histErr = models.ErrAccessDenied
	if _, err := api.GetHistory(ctx, &chatpb.GetHistoryRequest{ChatId: 3}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied")
	}
	// Not found
	api.chat.(*fakeChatService).histErr = models.ErrChatNotFound
	if _, err := api.GetHistory(ctx, &chatpb.GetHistoryRequest{ChatId: 3}); status.Code(err) != codes.NotFound {
		t.Fatalf("expected not found")
	}
	// Internal
	api.chat.(*fakeChatService).histErr = errors.New("db")
	if _, err := api.GetHistory(ctx, &chatpb.GetHistoryRequest{ChatId: 3}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal")
	}
	// Success
	api.chat.(*fakeChatService).histErr = nil
	api.chat.(*fakeChatService).histMsgs = []*chatpb.Message{{ChatId: 3, Text: "Hi"}}
	if resp, err := api.GetHistory(ctx, &chatpb.GetHistoryRequest{ChatId: 3}); err != nil || len(resp.Messages) != 1 {
		t.Fatalf("unexpected: %v %v", err, resp)
	}
}

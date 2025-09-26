package authgrpc

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"github.com/grigory222/go-chat-server/internal/domain/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeAuthService struct {
	loginRespAccess  string
	loginRespRefresh string
	loginUser        *chatpb.User
	loginErr         error
	regUser          *chatpb.User
	regErr           error
	refreshAccess    string
	refreshErr       error
}

func (f *fakeAuthService) Login(ctx context.Context, email, password string) (string, string, *chatpb.User, error) {
	return f.loginRespAccess, f.loginRespRefresh, f.loginUser, f.loginErr
}
func (f *fakeAuthService) Register(ctx context.Context, name, email, password string) (*chatpb.User, error) {
	return f.regUser, f.regErr
}
func (f *fakeAuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	return f.refreshAccess, f.refreshErr
}

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(os.Stdout, nil)) }

func TestAuthLoginHandler(t *testing.T) {
	api := &serverAPI{auth: &fakeAuthService{loginRespAccess: "a", loginRespRefresh: "r", loginUser: &chatpb.User{Id: 1}}, log: logger()}
	// Invalid args
	if _, err := api.Login(context.Background(), &chatpb.LoginRequest{Email: "", Password: ""}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument")
	}
	// Service user not found
	api.auth.(*fakeAuthService).loginErr = models.ErrUserNotFound
	if _, err := api.Login(context.Background(), &chatpb.LoginRequest{Email: "e", Password: "p"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated")
	}
	// Internal
	api.auth.(*fakeAuthService).loginErr = errors.New("db")
	if _, err := api.Login(context.Background(), &chatpb.LoginRequest{Email: "e", Password: "p"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal")
	}
	// Success
	api.auth.(*fakeAuthService).loginErr = nil
	if resp, err := api.Login(context.Background(), &chatpb.LoginRequest{Email: "e", Password: "p"}); err != nil || resp.AccessToken == "" {
		t.Fatalf("unexpected: %v %v", err, resp)
	}
}

func TestAuthRegisterHandler(t *testing.T) {
	api := &serverAPI{auth: &fakeAuthService{regUser: &chatpb.User{Id: 2}}, log: logger()}
	// Invalid
	if _, err := api.Register(context.Background(), &chatpb.RegisterRequest{Name: "", Email: "", Password: ""}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument")
	}
	// Exists
	api.auth.(*fakeAuthService).regErr = models.ErrUserExists
	if _, err := api.Register(context.Background(), &chatpb.RegisterRequest{Name: "n", Email: "e", Password: "p"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected already exists")
	}
	// Internal
	api.auth.(*fakeAuthService).regErr = errors.New("db")
	if _, err := api.Register(context.Background(), &chatpb.RegisterRequest{Name: "n", Email: "e", Password: "p"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal")
	}
	// Success
	api.auth.(*fakeAuthService).regErr = nil
	if resp, err := api.Register(context.Background(), &chatpb.RegisterRequest{Name: "n", Email: "e", Password: "p"}); err != nil || resp.User.Id == 0 {
		t.Fatalf("unexpected: %v %v", err, resp)
	}
}

func TestAuthRefreshHandler(t *testing.T) {
	api := &serverAPI{auth: &fakeAuthService{refreshAccess: "new"}, log: logger()}
	// Invalid
	if _, err := api.RefreshToken(context.Background(), &chatpb.RefreshTokenRequest{RefreshToken: ""}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument")
	}
	// Invalid credentials
	api.auth.(*fakeAuthService).refreshErr = models.ErrInvalidCredentials
	if _, err := api.RefreshToken(context.Background(), &chatpb.RefreshTokenRequest{RefreshToken: "x"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauth")
	}
	// Internal
	api.auth.(*fakeAuthService).refreshErr = errors.New("db")
	if _, err := api.RefreshToken(context.Background(), &chatpb.RefreshTokenRequest{RefreshToken: "x"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal")
	}
	// Success
	api.auth.(*fakeAuthService).refreshErr = nil
	if resp, err := api.RefreshToken(context.Background(), &chatpb.RefreshTokenRequest{RefreshToken: "x"}); err != nil || resp.AccessToken == "" {
		t.Fatalf("unexpected: %v %v", err, resp)
	}
}

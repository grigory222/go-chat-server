package interceptors

import (
	"context"
	"testing"
	"time"

	"log/slog"
	"os"

	"github.com/golang-jwt/jwt/v5"
	authsvc "github.com/grigory222/go-chat-server/internal/services/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(os.Stdout, nil)) }

func makeToken(secret []byte, uid int64, ttl time.Duration) string {
	claims := authsvc.Claims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))}, UserID: uid}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := tok.SignedString(secret)
	return s
}

func TestUnaryAuthInterceptor_PublicMethodsBypass(t *testing.T) {
	secret := "secret"
	interceptor := NewAuthInterceptor(logger(), secret)
	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) { called = true; return "ok", nil }
	// No metadata, but method is public
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/chat.AuthService/Login"}, handler)
	if err != nil || !called {
		t.Fatalf("public method should bypass auth: %v", err)
	}
}

func TestUnaryAuthInterceptor_ErrorsAndSuccess(t *testing.T) {
	secret := "secret"
	interceptor := NewAuthInterceptor(logger(), secret)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// Ensure user id exists
		if _, ok := ctx.Value(UserIDKey).(int64); !ok {
			t.Fatalf("user id not in context")
		}
		return "ok", nil
	}

	// Missing metadata
	if _, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/chat.ChatService/CreateChat"}, handler); err == nil {
		t.Fatalf("expected unauthenticated for missing metadata")
	}

	// Invalid header format
	md := metadata.New(map[string]string{"authorization": "Token abc"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	if _, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/chat.ChatService/CreateChat"}, handler); err == nil {
		t.Fatalf("expected invalid header format error")
	}

	// Invalid token
	md = metadata.New(map[string]string{"authorization": "Bearer invalid"})
	ctx = metadata.NewIncomingContext(context.Background(), md)
	if _, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/chat.ChatService/CreateChat"}, handler); err == nil {
		t.Fatalf("expected invalid token error")
	}

	// Valid token
	token := makeToken([]byte(secret), 77, time.Minute)
	md = metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewIncomingContext(context.Background(), md)
	if _, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/chat.ChatService/CreateChat"}, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Fake server stream
type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeServerStream) Context() context.Context { return f.ctx }

func TestStreamAuthInterceptor(t *testing.T) {
	secret := "secret"
	interceptor := NewAuthStreamInterceptor(logger(), secret)
	var gotUID int64
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		gotUID, _ = ss.Context().Value(UserIDKey).(int64)
		return nil
	}

	// Missing metadata
	if err := interceptor(nil, &fakeServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/chat.ChatService/JoinChat"}, handler); err == nil {
		t.Fatalf("expected error for missing metadata")
	}

	// Invalid header format
	md := metadata.New(map[string]string{"authorization": "Token abc"})
	if err := interceptor(nil, &fakeServerStream{ctx: metadata.NewIncomingContext(context.Background(), md)}, &grpc.StreamServerInfo{FullMethod: "/chat.ChatService/JoinChat"}, handler); err == nil {
		t.Fatalf("expected error invalid format")
	}

	// Invalid token
	md = metadata.New(map[string]string{"authorization": "Bearer invalid"})
	if err := interceptor(nil, &fakeServerStream{ctx: metadata.NewIncomingContext(context.Background(), md)}, &grpc.StreamServerInfo{FullMethod: "/chat.ChatService/JoinChat"}, handler); err == nil {
		t.Fatalf("expected error invalid token")
	}

	// Valid token
	token := makeToken([]byte(secret), 101, time.Minute)
	md = metadata.New(map[string]string{"authorization": "Bearer " + token})
	if err := interceptor(nil, &fakeServerStream{ctx: metadata.NewIncomingContext(context.Background(), md)}, &grpc.StreamServerInfo{FullMethod: "/chat.ChatService/JoinChat"}, handler); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if gotUID != 101 {
		t.Fatalf("expected uid 101, got %d", gotUID)
	}
}

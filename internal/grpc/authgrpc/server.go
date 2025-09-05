package authgrpc

import (
	"context"
	"github.com/grigory222/go-chat-server/internal/storage"
	"log/slog"
	"time"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"google.golang.org/grpc"
)

type serverAPI struct {
	chatpb.UnimplementedAuthServiceServer
	log             *slog.Logger
	storage         storage.Storage
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func Register(gRPC *grpc.Server, log *slog.Logger, storage storage.Storage, accessTokenTTL, refreshTokenTTL time.Duration) {
	chatpb.RegisterAuthServiceServer(gRPC, &serverAPI{
		log:             log,
		storage:         storage,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	})
}

func (s *serverAPI) Login(ctx context.Context, req *chatpb.LoginRequest) (*chatpb.LoginResponse, error) {
	return &chatpb.LoginResponse{AccessToken: "access_token"}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *chatpb.RegisterRequest) (*chatpb.RegisterResponse, error) {
	panic("implement me")
}

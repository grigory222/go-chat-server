package authgrpc

import (
	"context"

	chatpb "github.com/grigory222/go-chat-proto/gen/go/proto"
	"google.golang.org/grpc"
)

type serverAPI struct {
	chatpb.UnimplementedAuthServiceServer
}

func Register(gRPC *grpc.Server) {
	chatpb.RegisterAuthServiceServer(gRPC, &serverAPI{})
}

func (s *serverAPI) Login(ctx context.Context, req *chatpb.LoginRequest) (*chatpb.LoginResponse, error) {
	panic("implement me")
}

func (s *serverAPI) Register(ctx context.Context, req *chatpb.RegisterRequest) (*chatpb.RegisterResponse, error) {
	panic("implement me")
}

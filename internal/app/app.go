package app

import (
	"log/slog"

	grpcapp "github.com/grigory222/go-chat-server/internal/app/gprc"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int) *App {
	// storage
	// auth
	grpcApp := grpcapp.New(log, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}

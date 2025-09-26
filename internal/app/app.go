package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/grigory222/go-chat-server/internal/config"
	"github.com/grigory222/go-chat-server/internal/services/auth"
	"github.com/grigory222/go-chat-server/internal/services/chat"
	"github.com/grigory222/go-chat-server/internal/storage"
	"github.com/grigory222/go-chat-server/internal/storage/postgres"

	grpcapp "github.com/grigory222/go-chat-server/internal/app/gprc"
)

type App struct {
	GRPCSrv *grpcapp.App
	Storage storage.Storage
}

func New(log *slog.Logger, cfg *config.Config) *App {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pgStorage, err := postgres.New(ctx, cfg.Postgres, log)
	if err != nil {
		panic("failed to init storage: " + err.Error())
	}

	hub := chat.NewHub(log)

	authService := auth.New(log, pgStorage, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.JwtSecret)
	chatService := chat.New(log, pgStorage, hub)

	grpcApp := grpcapp.New(log, cfg.GRPC.Port, authService, chatService, cfg.JwtSecret)

	return &App{
		GRPCSrv: grpcApp,
		Storage: pgStorage,
	}
}

func (a *App) Stop() {
	a.GRPCSrv.Stop()
	a.Storage.Close()
}

package app

import (
	"context"
	"github.com/grigory222/go-chat-server/internal/config"
	"github.com/grigory222/go-chat-server/internal/services/auth"
	"github.com/grigory222/go-chat-server/internal/storage"
	"github.com/grigory222/go-chat-server/internal/storage/postgres"
	"log/slog"
	"time"

	grpcapp "github.com/grigory222/go-chat-server/internal/app/gprc"
)

type App struct {
	GRPCSrv *grpcapp.App
	Storage storage.Storage
}

func New(log *slog.Logger, grpcPort int, pgCfg config.Postgres, accessTokenTTL, refreshTokenTTL time.Duration, jwtSecret string) *App {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pgStorage, err := postgres.New(ctx, pgCfg, log)
	if err != nil {
		panic("failed to init storage: " + err.Error())
	}

	authService := auth.New(log, pgStorage, accessTokenTTL, refreshTokenTTL, jwtSecret)

	grpcApp := grpcapp.New(log, grpcPort, authService)

	return &App{
		GRPCSrv: grpcApp,
		Storage: pgStorage,
	}
}

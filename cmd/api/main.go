package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaitentenshii/laundry-pos-api/internal/app"
	"github.com/kaitentenshii/laundry-pos-api/internal/config"
	"github.com/kaitentenshii/laundry-pos-api/internal/store/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := postgres.Open(ctx, cfg.DatabaseURL, cfg.DatabaseConnectTimeout)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	logger.Info("PostgreSQL connection established")

	application := app.New(cfg, logger, database)
	if err := application.Run(ctx); err != nil {
		logger.Error("application stopped with an error", "error", err)
		os.Exit(1)
	}
}

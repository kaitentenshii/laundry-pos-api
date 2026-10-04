package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaitentenshii/laundry-pos-api/internal/app"
	"github.com/kaitentenshii/laundry-pos-api/internal/config"
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

	application := app.New(cfg, logger)
	if err := application.Run(ctx); err != nil {
		logger.Error("application stopped with an error", "error", err)
		os.Exit(1)
	}
}

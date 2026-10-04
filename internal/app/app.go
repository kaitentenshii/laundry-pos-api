package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/kaitentenshii/laundry-pos-api/internal/config"
	"github.com/kaitentenshii/laundry-pos-api/internal/httpapi"
)

type App struct {
	config config.Config
	logger *slog.Logger
	server *http.Server
}

func New(cfg config.Config, logger *slog.Logger, readinessChecker httpapi.ReadinessChecker) *App {
	return &App{
		config: cfg,
		logger: logger,
		server: &http.Server{
			Addr:              cfg.Address(),
			Handler:           httpapi.NewRouter(logger, readinessChecker),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (a *App) Run(ctx context.Context) error {
	serverErrors := make(chan error, 1)

	go func() {
		a.logger.Info("HTTP server started", "address", a.server.Addr, "environment", a.config.Environment)
		serverErrors <- a.server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		a.logger.Info("shutting down HTTP server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.config.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return errors.Join(err, a.server.Close())
	}

	return nil
}

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/kaitentenshii/laundry-pos-api/internal/config"
	dbmigration "github.com/kaitentenshii/laundry-pos-api/internal/migration"
)

const usage = "usage: go run ./cmd/migrate <up|down> [steps]"

type command struct {
	action string
	steps  int
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(os.Args[1:]); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	logger.Info("database migration completed", "action", os.Args[1])
}

func run(args []string) (runErr error) {
	command, err := parseCommand(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	runner, err := dbmigration.New(cfg.MigrationsPath, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := runner.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close database migration: %w", err))
		}
	}()

	switch command.action {
	case "up":
		return runner.Up()
	case "down":
		return runner.Down(command.steps)
	default:
		return fmt.Errorf("unsupported migration action %q", command.action)
	}
}

func parseCommand(args []string) (command, error) {
	if len(args) == 0 {
		return command{}, errors.New(usage)
	}

	switch args[0] {
	case "up":
		if len(args) != 1 {
			return command{}, errors.New(usage)
		}
		return command{action: "up"}, nil
	case "down":
		if len(args) > 2 {
			return command{}, errors.New(usage)
		}

		steps := 1
		if len(args) == 2 {
			parsedSteps, err := strconv.Atoi(args[1])
			if err != nil || parsedSteps < 1 {
				return command{}, errors.New("migration rollback steps must be a positive number")
			}
			steps = parsedSteps
		}

		return command{action: "down", steps: steps}, nil
	default:
		return command{}, errors.New(usage)
	}
}

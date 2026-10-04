package migration

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Runner struct {
	engine *migrate.Migrate
}

func New(sourceURL, databaseURL string) (*Runner, error) {
	engine, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("initialize database migration: %w", err)
	}

	return &Runner{engine: engine}, nil
}

func (r *Runner) Up() error {
	if err := r.engine.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply database migrations: %w", err)
	}

	return nil
}

func (r *Runner) Down(steps int) error {
	if steps < 1 {
		return errors.New("rollback steps must be greater than zero")
	}

	if err := r.engine.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("rollback database migrations: %w", err)
	}

	return nil
}

func (r *Runner) Close() error {
	sourceErr, databaseErr := r.engine.Close()
	return errors.Join(sourceErr, databaseErr)
}

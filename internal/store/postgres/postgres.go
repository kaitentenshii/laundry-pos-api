package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, databaseURL string, connectTimeout time.Duration) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("DATABASE_URL is invalid")
	}

	poolConfig.ConnConfig.ConnectTimeout = connectTimeout
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL connection pool: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := pool.Ping(pingContext); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return pool, nil
}

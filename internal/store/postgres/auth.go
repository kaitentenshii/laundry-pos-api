package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type AuthStore struct{ pool *pgxpool.Pool }

func NewAuthStore(pool *pgxpool.Pool) *AuthStore { return &AuthStore{pool: pool} }

func scanUser(row pgx.Row) (service.User, error) {
	var user service.User
	err := row.Scan(&user.ID, &user.Name, &user.Username, &user.Email, &user.Role, &user.PasswordHash, &user.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.User{}, service.ErrNotFound
	}
	return user, err
}

func (s *AuthStore) FindUser(ctx context.Context, identifier string) (service.User, error) {
	// Usernames cannot contain @, so username and email identifiers are unambiguous.
	column := "username"
	if strings.Contains(identifier, "@") {
		column = "email"
	}
	return scanUser(s.pool.QueryRow(ctx, `SELECT id, name, username, email, role, password_hash, is_active FROM laundry.users WHERE lower(`+column+`) = lower($1)`, identifier))
}

func (s *AuthStore) CreateSession(ctx context.Context, hash []byte, userID int64, expiry time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO laundry.auth_sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`, hash, userID, expiry)
	return err
}

func (s *AuthStore) SessionUser(ctx context.Context, hash []byte, now time.Time) (service.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT u.id, u.name, u.username, u.email, u.role, u.password_hash, u.is_active FROM laundry.auth_sessions s JOIN laundry.users u ON u.id = s.user_id WHERE s.token_hash = $1 AND s.expires_at > $2`, hash, now))
}

func (s *AuthStore) DeleteSession(ctx context.Context, hash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM laundry.auth_sessions WHERE token_hash = $1`, hash)
	return err
}

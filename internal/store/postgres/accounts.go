package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type AccountStore struct{ pool *pgxpool.Pool }

func NewAccountStore(pool *pgxpool.Pool) *AccountStore { return &AccountStore{pool: pool} }

const accountColumns = `id,name,username,COALESCE(email,''),role,is_active,last_active_at`

func scanAccount(row pgx.Row) (service.Account, error) {
	var a service.Account
	err := row.Scan(&a.ID, &a.Name, &a.Username, &a.Email, &a.Role, &a.IsActive, &a.LastActiveAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, service.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return a, service.ErrConflict
	}
	return a, err
}
func (s *AccountStore) ListAccounts(ctx context.Context) ([]service.Account, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+accountColumns+` FROM laundry.users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]service.Account, 0)
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
func (s *AccountStore) InsertAccount(ctx context.Context, a service.Account, hash string) (service.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx, `INSERT INTO laundry.users(name,username,email,role,is_active,password_hash) VALUES($1,$2,NULLIF($3,''),$4,$5,$6) RETURNING `+accountColumns, a.Name, a.Username, a.Email, a.Role, a.IsActive, hash))
}
func (s *AccountStore) UpdateAccount(ctx context.Context, actorID, id int64, a service.Account) (service.Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return service.Account{}, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id FROM laundry.users WHERE role='admin' AND is_active ORDER BY id FOR UPDATE`)
	if err != nil {
		return service.Account{}, err
	}
	admins := 0
	for rows.Next() {
		admins++
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return service.Account{}, err
	}
	old, err := scanAccount(tx.QueryRow(ctx, `SELECT `+accountColumns+` FROM laundry.users WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return service.Account{}, err
	}
	if id == actorID && (!a.IsActive || a.Role != "admin") {
		return service.Account{}, service.ErrSelfLockout
	}
	if old.IsActive && old.Role == "admin" && (!a.IsActive || a.Role != "admin") && admins <= 1 {
		return service.Account{}, service.ErrLastAdmin
	}
	if a.Username == "" {
		a.Username = old.Username
	}
	updated, err := scanAccount(tx.QueryRow(ctx, `UPDATE laundry.users SET name=$2,username=$3,email=NULLIF($4,''),role=$5,is_active=$6 WHERE id=$1 RETURNING `+accountColumns, id, a.Name, a.Username, a.Email, a.Role, a.IsActive))
	if err != nil {
		return service.Account{}, err
	}
	if !a.IsActive || a.Role != old.Role {
		if _, err := tx.Exec(ctx, `DELETE FROM laundry.auth_sessions WHERE user_id=$1`, id); err != nil {
			return service.Account{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return service.Account{}, err
	}
	return updated, nil
}

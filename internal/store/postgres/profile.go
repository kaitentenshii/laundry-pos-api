package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type ProfileStore struct{ pool *pgxpool.Pool }

func NewProfileStore(pool *pgxpool.Pool) *ProfileStore { return &ProfileStore{pool} }
func (s *ProfileStore) GetProfile(ctx context.Context) (service.LaundryProfile, error) {
	var p service.LaundryProfile
	err := s.pool.QueryRow(ctx, `SELECT name,contact,operating_hours,address,updated_at FROM laundry.profile WHERE id=true`).Scan(&p.Name, &p.Contact, &p.OperatingHours, &p.Address, &p.UpdatedAt)
	return p, err
}
func (s *ProfileStore) SaveProfile(ctx context.Context, input service.ProfileInput) (service.LaundryProfile, error) {
	var p service.LaundryProfile
	err := s.pool.QueryRow(ctx, `UPDATE laundry.profile SET name=$1,contact=$2,operating_hours=$3,address=$4,updated_at=now() WHERE id=true RETURNING name,contact,operating_hours,address,updated_at`, input.Name, input.Contact, input.OperatingHours, input.Address).Scan(&p.Name, &p.Contact, &p.OperatingHours, &p.Address, &p.UpdatedAt)
	return p, err
}

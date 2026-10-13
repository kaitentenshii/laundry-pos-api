package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type LaundryServiceStore struct{ pool *pgxpool.Pool }

func NewLaundryServiceStore(pool *pgxpool.Pool) *LaundryServiceStore {
	return &LaundryServiceStore{pool}
}

const serviceColumns = `id,name,unit,price,estimated_hours,is_active,created_at,updated_at`

func scanService(row pgx.Row) (service.LaundryService, error) {
	var data service.LaundryService
	err := row.Scan(&data.ID, &data.Name, &data.Unit, &data.Price, &data.EstimatedHours, &data.IsActive, &data.CreatedAt, &data.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return data, service.ErrNotFound
	}
	return data, err
}
func (s *LaundryServiceStore) ListServices(ctx context.Context, activeOnly bool) ([]service.LaundryService, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+serviceColumns+` FROM laundry.services WHERE (NOT $1::boolean OR is_active) ORDER BY id`, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]service.LaundryService, 0)
	for rows.Next() {
		data, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, data)
	}
	return result, rows.Err()
}
func (s *LaundryServiceStore) SaveService(ctx context.Context, id int64, data service.LaundryService) (service.LaundryService, error) {
	if id == 0 {
		return scanService(s.pool.QueryRow(ctx, `INSERT INTO laundry.services(name,unit,price,estimated_hours,is_active) VALUES($1,$2,$3,$4,$5) RETURNING `+serviceColumns, data.Name, data.Unit, data.Price, data.EstimatedHours, data.IsActive))
	}
	return scanService(s.pool.QueryRow(ctx, `UPDATE laundry.services SET name=$2,unit=$3,price=$4,estimated_hours=$5,is_active=$6,updated_at=now() WHERE id=$1 RETURNING `+serviceColumns, id, data.Name, data.Unit, data.Price, data.EstimatedHours, data.IsActive))
}

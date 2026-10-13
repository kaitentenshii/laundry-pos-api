package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type LaundryServiceStore struct{ pool *pgxpool.Pool }

func NewLaundryServiceStore(pool *pgxpool.Pool) *LaundryServiceStore {
	return &LaundryServiceStore{pool}
}

const serviceColumns = `id,name,unit,price,estimated_hours,is_active,created_at,updated_at,COALESCE(price_version_id,0)`

func scanService(row pgx.Row) (service.LaundryService, error) {
	var data service.LaundryService
	err := row.Scan(&data.ID, &data.Name, &data.Unit, &data.Price, &data.EstimatedHours, &data.IsActive, &data.CreatedAt, &data.UpdatedAt, &data.PriceVersionID)
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return service.LaundryService{}, err
	}
	defer tx.Rollback(ctx)
	if id == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO laundry.services(name,unit,price,estimated_hours,is_active) VALUES($1,$2,$3,$4,$5) RETURNING id`, data.Name, data.Unit, data.Price, data.EstimatedHours, data.IsActive).Scan(&id)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE laundry.services SET name=$2,unit=$3,price=$4,estimated_hours=$5,is_active=$6,updated_at=now() WHERE id=$1`, id, data.Name, data.Unit, data.Price, data.EstimatedHours, data.IsActive)
		if err == nil && tag.RowsAffected() == 0 {
			return service.LaundryService{}, service.ErrNotFound
		}
	}
	if err != nil {
		return service.LaundryService{}, err
	}
	result, err := scanService(tx.QueryRow(ctx, `SELECT `+serviceColumns+` FROM laundry.services WHERE id=$1`, id))
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

func (s *LaundryServiceStore) ListPriceVersions(ctx context.Context, id int64) ([]service.ServicePriceVersion, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM laundry.services WHERE id=$1)`, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, service.ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT id,service_id,name,unit,price,estimated_hours,created_at FROM laundry.service_price_versions WHERE service_id=$1 ORDER BY id DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	data := make([]service.ServicePriceVersion, 0)
	for rows.Next() {
		var version service.ServicePriceVersion
		if err := rows.Scan(&version.ID, &version.ServiceID, &version.Name, &version.Unit, &version.Price, &version.EstimatedHours, &version.CreatedAt); err != nil {
			return nil, err
		}
		data = append(data, version)
	}
	return data, rows.Err()
}

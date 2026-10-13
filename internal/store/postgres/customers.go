package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type CustomerStore struct{ pool *pgxpool.Pool }

func NewCustomerStore(pool *pgxpool.Pool) *CustomerStore { return &CustomerStore{pool} }

const customerColumns = `id,name,whatsapp,address,notes,whatsapp_opt_in,created_at,updated_at`

func scanCustomer(row pgx.Row) (service.Customer, error) {
	var c service.Customer
	err := row.Scan(&c.ID, &c.Name, &c.Whatsapp, &c.Address, &c.Notes, &c.WhatsappOptIn, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}
func (s *CustomerStore) ListCustomers(ctx context.Context) ([]service.Customer, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+customerColumns+` FROM laundry.customers ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	data := make([]service.Customer, 0)
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		data = append(data, c)
	}
	return data, rows.Err()
}
func (s *CustomerStore) InsertCustomer(ctx context.Context, input service.CustomerInput) (service.Customer, error) {
	return scanCustomer(s.pool.QueryRow(ctx, `INSERT INTO laundry.customers(name,whatsapp,address,notes,whatsapp_opt_in) VALUES($1,$2,$3,$4,$5) RETURNING `+customerColumns, input.Name, input.Whatsapp, input.Address, input.Notes, input.WhatsappOptIn))
}

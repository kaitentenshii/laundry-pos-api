package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"strings"
)

type CustomerStore struct{ pool *pgxpool.Pool }

func NewCustomerStore(pool *pgxpool.Pool) *CustomerStore { return &CustomerStore{pool} }

func (s *CustomerStore) GetCustomer(ctx context.Context, id int64) (service.Customer, error) {
	return scanCustomer(s.pool.QueryRow(ctx, `SELECT `+customerColumns+` FROM laundry.customers WHERE id=$1`, id))
}

const customerColumns = `id,name,whatsapp,address,notes,whatsapp_opt_in,created_at,updated_at,(SELECT count(*) FROM laundry.orders WHERE customer_id=laundry.customers.id)`

func scanCustomer(row pgx.Row) (service.Customer, error) {
	var c service.Customer
	err := row.Scan(&c.ID, &c.Name, &c.Whatsapp, &c.Address, &c.Notes, &c.WhatsappOptIn, &c.CreatedAt, &c.UpdatedAt, &c.OrderCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, service.ErrNotFound
	}
	return c, err
}

func (s *CustomerStore) UpdateCustomer(ctx context.Context, id int64, input service.CustomerInput) (service.Customer, error) {
	return scanCustomer(s.pool.QueryRow(ctx, `UPDATE laundry.customers SET name=$2,whatsapp=$3,address=$4,notes=$5,whatsapp_opt_in=$6,updated_at=now() WHERE id=$1 RETURNING `+customerColumns, id, input.Name, input.Whatsapp, input.Address, input.Notes, input.WhatsappOptIn))
}
func (s *CustomerStore) ListCustomers(ctx context.Context, q service.CustomerQuery) (service.CustomerList, error) {
	result := service.CustomerList{Data: make([]service.Customer, 0), Page: q.Page, PageSize: q.PageSize}
	escape := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)
	phone := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(q.Search)
	if strings.HasPrefix(phone, "08") {
		phone = "+62" + phone[1:]
	} else if strings.HasPrefix(phone, "62") {
		phone = "+" + phone
	}
	namePattern, phonePattern := "%"+escape.Replace(q.Search)+"%", "%"+escape.Replace(phone)+"%"
	const filter = ` WHERE (name ILIKE $1 OR whatsapp LIKE $2) AND ($3::boolean IS NULL OR whatsapp_opt_in=$3)`
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM laundry.customers`+filter, namePattern, phonePattern, q.Consent).Scan(&result.Total); err != nil {
		return result, err
	}
	rows, err := tx.Query(ctx, `SELECT `+customerColumns+` FROM laundry.customers`+filter+` ORDER BY id DESC LIMIT $4 OFFSET $5`, namePattern, phonePattern, q.Consent, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return result, err
		}
		result.Data = append(result.Data, c)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	rows.Close()
	return result, tx.Commit(ctx)
}
func (s *CustomerStore) InsertCustomer(ctx context.Context, input service.CustomerInput) (service.Customer, error) {
	return scanCustomer(s.pool.QueryRow(ctx, `INSERT INTO laundry.customers(name,whatsapp,address,notes,whatsapp_opt_in) VALUES($1,$2,$3,$4,$5) RETURNING `+customerColumns, input.Name, input.Whatsapp, input.Address, input.Notes, input.WhatsappOptIn))
}

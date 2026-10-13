package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"sort"
)

type OrderStore struct{ pool *pgxpool.Pool }

func NewOrderStore(pool *pgxpool.Pool) *OrderStore { return &OrderStore{pool} }

type orderQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

const orderColumns = `o.id,o.order_number,o.customer_id,o.customer_name,o.customer_whatsapp,o.customer_opt_in,o.created_by,u.name,o.due_at,o.notes,o.status,o.payment_status,o.total,o.created_at,o.updated_at,(SELECT COALESCE(sum(p.amount_settled),0) FROM laundry.payments p WHERE p.order_id=o.id)`

func scanOrder(row pgx.Row) (service.Order, error) {
	var o service.Order
	err := row.Scan(&o.ID, &o.Number, &o.CustomerID, &o.CustomerName, &o.CustomerWhatsapp, &o.CustomerOptIn, &o.CreatedBy, &o.CreatorName, &o.DueAt, &o.Notes, &o.Status, &o.PaymentStatus, &o.Total, &o.CreatedAt, &o.UpdatedAt, &o.PaidAmount)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, service.ErrNotFound
	}
	o.Balance = o.Total - o.PaidAmount
	o.AllowedNextStatuses = service.AllowedOrderTransitions(o.Status, o.PaymentStatus)
	if o.Status == "COMPLETED" && o.PaymentStatus != "PAID" {
		o.PickupBlockedReason = "Lunasi pembayaran sebelum menyerahkan cucian kepada pelanggan."
	}
	return o, err
}
func readOrder(ctx context.Context, q orderQuerier, id int64) (service.Order, error) {
	o, err := scanOrder(q.QueryRow(ctx, `SELECT `+orderColumns+` FROM laundry.orders o JOIN laundry.users u ON u.id=o.created_by WHERE o.id=$1`, id))
	if err != nil {
		return o, err
	}
	rows, err := q.Query(ctx, `SELECT id,service_id,price_version_id,service_name,unit,unit_price,quantity_milli,subtotal FROM laundry.order_items WHERE order_id=$1 ORDER BY id`, id)
	if err != nil {
		return o, err
	}
	defer rows.Close()
	o.Items = make([]service.OrderItem, 0)
	for rows.Next() {
		var item service.OrderItem
		if err := rows.Scan(&item.ID, &item.ServiceID, &item.PriceVersionID, &item.ServiceName, &item.Unit, &item.UnitPrice, &item.QuantityMilli, &item.Subtotal); err != nil {
			return o, err
		}
		o.Items = append(o.Items, item)
	}
	if err := rows.Err(); err != nil {
		return o, err
	}
	rows.Close()
	payment, err := readPayment(ctx, q, id)
	if err == nil {
		o.Payment = &payment
		o.PaidAmount = payment.AmountSettled
		o.Balance = o.Total - o.PaidAmount
	} else if !errors.Is(err, service.ErrNotFound) {
		return o, err
	}
	return o, nil
}
func (s *OrderStore) GetOrder(ctx context.Context, id int64) (service.Order, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return service.Order{}, err
	}
	defer tx.Rollback(ctx)
	o, err := readOrder(ctx, tx, id)
	if err != nil {
		return o, err
	}
	return o, tx.Commit(ctx)
}
func (s *OrderStore) InsertOrder(ctx context.Context, actor int64, input service.OrderInput, hash string) (service.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return service.Order{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("order:%d:%s", actor, input.RequestKey)); err != nil {
		return service.Order{}, err
	}
	var existingID int64
	var existingHash string
	err = tx.QueryRow(ctx, `SELECT id,request_hash FROM laundry.orders WHERE created_by=$1 AND request_key=$2`, actor, input.RequestKey).Scan(&existingID, &existingHash)
	if err == nil {
		if hash != existingHash {
			return service.Order{}, service.Invalid("request_key", "Permintaan ini sudah digunakan untuk data lain.")
		}
		o, err := readOrder(ctx, tx, existingID)
		if err != nil {
			return o, err
		}
		return o, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return service.Order{}, err
	}
	customer, err := scanCustomer(tx.QueryRow(ctx, `SELECT `+customerColumns+` FROM laundry.customers WHERE id=$1 FOR SHARE`, input.CustomerID))
	if errors.Is(err, service.ErrNotFound) {
		return service.Order{}, service.Invalid("customer_id", "Pelanggan tidak ditemukan.")
	}
	if err != nil {
		return service.Order{}, err
	}
	ids := make([]int64, 0)
	seen := map[int64]bool{}
	for _, item := range input.Items {
		if !seen[item.ServiceID] {
			seen[item.ServiceID] = true
			ids = append(ids, item.ServiceID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	services := map[int64]service.LaundryService{}
	for _, id := range ids {
		selected, err := scanService(tx.QueryRow(ctx, `SELECT `+serviceColumns+` FROM laundry.services WHERE id=$1 FOR SHARE`, id))
		if errors.Is(err, service.ErrNotFound) || err == nil && !selected.IsActive {
			return service.Order{}, service.Invalid("items", "Layanan tidak tersedia atau sudah nonaktif. Muat ulang pilihan layanan.")
		}
		if err != nil {
			return service.Order{}, err
		}
		services[id] = selected
	}
	items := make([]service.OrderItem, 0, len(input.Items))
	var total int64
	for _, line := range input.Items {
		selected := services[line.ServiceID]
		milli, err := service.ParseQuantity(line.Quantity, selected.Unit)
		if err != nil {
			return service.Order{}, err
		}
		subtotal, err := service.ItemSubtotal(selected.Price, milli)
		if err != nil {
			return service.Order{}, err
		}
		total += subtotal
		if total > service.MaxOrderTotal {
			return service.Order{}, service.Invalid("items", "Total melebihi batas transaksi.")
		}
		items = append(items, service.OrderItem{ServiceID: selected.ID, PriceVersionID: selected.PriceVersionID, ServiceName: selected.Name, Unit: selected.Unit, UnitPrice: selected.Price, QuantityMilli: milli, Subtotal: subtotal})
	}
	var id int64
	if input.ExpectedTotal != nil && total != *input.ExpectedTotal {
		return service.Order{}, service.Invalid("expected_total", "Harga layanan berubah. Periksa ringkasan terbaru sebelum menyimpan kembali.")
	}
	err = tx.QueryRow(ctx, `INSERT INTO laundry.orders(customer_id,customer_name,customer_whatsapp,customer_opt_in,created_by,due_at,notes,total,request_key,request_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`, customer.ID, customer.Name, customer.Whatsapp, customer.WhatsappOptIn, actor, input.DueAt, input.Notes, total, input.RequestKey, hash).Scan(&id)
	if err != nil {
		return service.Order{}, err
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `INSERT INTO laundry.order_items(order_id,service_id,price_version_id,service_name,unit,unit_price,quantity_milli,subtotal) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, item.ServiceID, item.PriceVersionID, item.ServiceName, item.Unit, item.UnitPrice, item.QuantityMilli, item.Subtotal); err != nil {
			return service.Order{}, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO laundry.order_status_history(order_id,to_status,changed_by,actor_name,changed_at,request_key) SELECT o.id,'RECEIVED',o.created_by,u.name,o.created_at,'created:'||o.id::text FROM laundry.orders o JOIN laundry.users u ON u.id=o.created_by WHERE o.id=$1`, id); err != nil {
		return service.Order{}, err
	}
	o, err := readOrder(ctx, tx, id)
	if err != nil {
		return o, err
	}
	return o, tx.Commit(ctx)
}

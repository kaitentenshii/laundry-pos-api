package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"strings"
)

func (s *OrderStore) ListOrders(ctx context.Context, q service.OrderQuery) (service.OrderList, error) {
	result := service.OrderList{Data: make([]service.Order, 0), Page: q.Page, PageSize: q.PageSize}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q.Search) + "%"
	const filter = ` WHERE ($1='' OR o.status=$1) AND ($2='' OR o.payment_status=$2) AND ($3::timestamptz IS NULL OR o.created_at >= $3) AND ($4::timestamptz IS NULL OR o.created_at < $4) AND (o.order_number ILIKE $5 OR o.customer_name ILIKE $5)`
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM laundry.orders o`+filter, q.Status, q.PaymentStatus, q.From, q.To, pattern).Scan(&result.Total); err != nil {
		return result, err
	}
	rows, err := tx.Query(ctx, `SELECT `+orderColumns+` FROM laundry.orders o JOIN laundry.users u ON u.id=o.created_by`+filter+` ORDER BY o.id DESC LIMIT $6 OFFSET $7`, q.Status, q.PaymentStatus, q.From, q.To, pattern, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return result, err
	}
	ids := make([]int64, 0)
	positions := map[int64]int{}
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			rows.Close()
			return result, err
		}
		order.Items = make([]service.OrderItem, 0)
		positions[order.ID] = len(result.Data)
		ids = append(ids, order.ID)
		result.Data = append(result.Data, order)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	if len(ids) > 0 {
		items, err := tx.Query(ctx, `SELECT order_id,id,service_id,price_version_id,service_name,unit,unit_price,quantity_milli,subtotal FROM laundry.order_items WHERE order_id=ANY($1) ORDER BY id`, ids)
		if err != nil {
			return result, err
		}
		for items.Next() {
			var orderID int64
			var item service.OrderItem
			if err := items.Scan(&orderID, &item.ID, &item.ServiceID, &item.PriceVersionID, &item.ServiceName, &item.Unit, &item.UnitPrice, &item.QuantityMilli, &item.Subtotal); err != nil {
				items.Close()
				return result, err
			}
			index := positions[orderID]
			result.Data[index].Items = append(result.Data[index].Items, item)
		}
		err = items.Err()
		items.Close()
		if err != nil {
			return result, err
		}
	}
	return result, tx.Commit(ctx)
}

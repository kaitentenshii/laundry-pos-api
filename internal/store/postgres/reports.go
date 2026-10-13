package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

func (s *OrderStore) Report(ctx context.Context, q service.ReportQuery) (service.TransactionReport, error) {
	result := service.TransactionReport{Orders: make([]service.Order, 0), Page: q.Page, PageSize: q.PageSize, From: q.FromDate, To: q.ToDate, Timezone: "Asia/Jakarta"}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT count(*),COALESCE(sum(o.total),0),COALESCE(sum(o.total-COALESCE(p.settled,0)),0) FROM laundry.orders o LEFT JOIN (SELECT order_id,sum(amount_settled) settled FROM laundry.payments GROUP BY order_id) p ON p.order_id=o.id WHERE o.created_at >= $1 AND o.created_at < $2`, q.From, q.To).Scan(&result.Total, &result.OrderValue, &result.Balance)
	if err != nil {
		return result, err
	}
	err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_settled),0) FROM laundry.payments WHERE paid_at >= $1 AND paid_at < $2`, q.From, q.To).Scan(&result.Receipts)
	if err != nil {
		return result, err
	}
	rows, err := tx.Query(ctx, `SELECT `+orderColumns+` FROM laundry.orders o JOIN laundry.users u ON u.id=o.created_by WHERE o.created_at >= $1 AND o.created_at < $2 ORDER BY o.created_at DESC,o.id DESC LIMIT $3 OFFSET $4`, q.From, q.To, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		o, e := scanOrder(rows)
		if e != nil {
			return result, e
		}
		o.Items = make([]service.OrderItem, 0)
		result.Orders = append(result.Orders, o)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

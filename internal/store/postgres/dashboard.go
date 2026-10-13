package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"time"
)

func (s *OrderStore) Dashboard(ctx context.Context, now time.Time) (service.Dashboard, error) {
	zone := time.FixedZone("Asia/Jakarta", 7*60*60)
	local := now.In(zone)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	end := start.AddDate(0, 0, 1)
	result := service.Dashboard{AsOf: now, BusinessDate: start.Format("2006-01-02"), Timezone: "Asia/Jakarta", Monitored: make([]service.Order, 0)}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `SELECT count(*) FILTER(WHERE status IN ('RECEIVED','WASHING')),count(*) FILTER(WHERE status='WASHING'),count(*) FILTER(WHERE status='RECEIVED'),count(*) FILTER(WHERE status='COMPLETED'),count(*) FILTER(WHERE status IN ('RECEIVED','WASHING') AND due_at<$1) FROM laundry.orders`, now).Scan(&result.Active, &result.Washing, &result.Waiting, &result.Ready, &result.Overdue)
	if err != nil {
		return result, err
	}
	err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount_settled),0),count(*) FROM laundry.payments WHERE paid_at >= $1 AND paid_at < $2`, start, end).Scan(&result.Receipts, &result.PaymentCount)
	if err != nil {
		return result, err
	}
	rows, err := tx.Query(ctx, `SELECT o.id FROM laundry.orders o WHERE status IN ('RECEIVED','WASHING','COMPLETED') ORDER BY CASE WHEN status IN ('RECEIVED','WASHING') AND due_at<$1 THEN 0 WHEN status='COMPLETED' THEN 1 ELSE 2 END,due_at,id LIMIT 8`, now)
	if err != nil {
		return result, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return result, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, id := range ids {
		order, e := readOrder(ctx, tx, id)
		if e != nil {
			return result, e
		}
		result.Monitored = append(result.Monitored, order)
	}
	return result, tx.Commit(ctx)
}

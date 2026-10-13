package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"time"
)

func (s *OrderStore) TransitionOrder(ctx context.Context, actor, id int64, input service.OrderStatusInput) (service.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return service.Order{}, err
	}
	defer tx.Rollback(ctx)
	var current, payment string
	err = tx.QueryRow(ctx, `SELECT status,payment_status FROM laundry.orders WHERE id=$1 FOR UPDATE`, id).Scan(&current, &payment)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Order{}, service.ErrNotFound
	}
	if err != nil {
		return service.Order{}, err
	}
	var oldFrom, oldTo string
	var oldActor int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(from_status,''),to_status,changed_by FROM laundry.order_status_history WHERE order_id=$1 AND request_key=$2`, id, input.RequestKey).Scan(&oldFrom, &oldTo, &oldActor)
	if err == nil {
		if oldFrom != input.FromStatus || oldTo != input.Status || oldActor != actor {
			return service.Order{}, service.Invalid("request_key", "Kunci permintaan telah dipakai untuk perubahan lain.")
		}
		o, err := readOrder(ctx, tx, id)
		if err != nil {
			return o, err
		}
		return o, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return service.Order{}, err
	}
	if current != input.FromStatus {
		return service.Order{}, service.ErrStaleOrder
	}
	if err := service.ValidateOrderTransition(current, input.Status, payment); err != nil {
		return service.Order{}, err
	}
	changedAt := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE laundry.orders SET status=$2,updated_at=$3 WHERE id=$1`, id, input.Status, changedAt); err != nil {
		return service.Order{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO laundry.order_status_history(order_id,from_status,to_status,changed_by,actor_name,changed_at,request_key) SELECT $1,$2,$3,$4,name,$5,$6 FROM laundry.users WHERE id=$4`, id, current, input.Status, actor, changedAt, input.RequestKey); err != nil {
		return service.Order{}, err
	}
	o, err := readOrder(ctx, tx, id)
	if err != nil {
		return o, err
	}
	return o, tx.Commit(ctx)
}

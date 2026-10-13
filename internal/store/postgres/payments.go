package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"time"
)

func readPayment(ctx context.Context, q orderQuerier, id int64) (service.Payment, error) {
	var p service.Payment
	err := q.QueryRow(ctx, `SELECT p.id,p.order_id,o.order_number,p.method,p.amount_received,p.amount_settled,p.change_amount,p.paid_at,p.created_by,u.name,p.notes,p.created_at FROM laundry.payments p JOIN laundry.orders o ON o.id=p.order_id JOIN laundry.users u ON u.id=p.created_by WHERE p.order_id=$1`, id).Scan(&p.ID, &p.OrderID, &p.OrderNumber, &p.Method, &p.AmountReceived, &p.AmountSettled, &p.ChangeAmount, &p.PaidAt, &p.CreatedBy, &p.CreatorName, &p.Notes, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, service.ErrNotFound
	}
	return p, err
}
func (s *OrderStore) InsertPayment(ctx context.Context, actor, id int64, input service.PaymentInput, hash string) (service.Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return service.Payment{}, err
	}
	defer tx.Rollback(ctx)
	var total int64
	var paymentStatus string
	var createdAt time.Time
	err = tx.QueryRow(ctx, `SELECT total,payment_status,created_at FROM laundry.orders WHERE id=$1 FOR UPDATE`, id).Scan(&total, &paymentStatus, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Payment{}, service.ErrNotFound
	}
	if err != nil {
		return service.Payment{}, err
	}
	var existingKey, existingHash string
	var existingActor int64
	err = tx.QueryRow(ctx, `SELECT request_key,request_hash,created_by FROM laundry.payments WHERE order_id=$1`, id).Scan(&existingKey, &existingHash, &existingActor)
	if err == nil {
		if existingKey != input.RequestKey || existingHash != hash || existingActor != actor {
			return service.Payment{}, service.ErrAlreadyPaid
		}
		p, err := readPayment(ctx, tx, id)
		if err != nil {
			return p, err
		}
		return p, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return service.Payment{}, err
	}
	if paymentStatus == "PAID" {
		return service.Payment{}, service.ErrAlreadyPaid
	}
	if *input.AmountReceived < total {
		return service.Payment{}, service.Invalid("amount_received", "Nominal harus cukup untuk melunasi seluruh sisa tagihan.")
	}
	if input.PaidAt.Before(createdAt.Add(-time.Minute)) {
		return service.Payment{}, service.Invalid("paid_at", "Pembayaran tidak boleh sebelum pesanan dibuat.")
	}
	change := int64(0)
	if input.Method == "CASH" {
		change = *input.AmountReceived - total
	}
	if _, err := tx.Exec(ctx, `INSERT INTO laundry.payments(order_id,method,amount_received,amount_settled,change_amount,paid_at,created_by,notes,confirmed,request_key,request_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, input.Method, *input.AmountReceived, total, change, input.PaidAt, actor, input.Notes, input.Confirmed, input.RequestKey, hash); err != nil {
		return service.Payment{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE laundry.orders SET payment_status='PAID',updated_at=now() WHERE id=$1`, id); err != nil {
		return service.Payment{}, err
	}
	p, err := readPayment(ctx, tx, id)
	if err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}

package postgres

import (
	"context"
	"errors"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"sync"
	"testing"
	"time"
)

func TestManualPaymentPostgres(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	order, err := f.manager.Create(ctx, f.actor, f.input("payment_order_123456789"))
	if err != nil {
		t.Fatal(err)
	}
	received := int64(31000)
	input := service.PaymentInput{Method: "CASH", AmountReceived: &received, PaidAt: time.Now(), Notes: "Cash paid", RequestKey: "payment_request_1234567"}
	if _, err := f.manager.Pay(ctx, f.actor, order.ID, input); err == nil {
		t.Fatal("partial payment accepted")
	}
	received = 50000
	payment, err := f.manager.Pay(ctx, f.actor, order.ID, input)
	if err != nil || payment.ChangeAmount != 18000 || payment.AmountSettled != 32000 || payment.CreatorName != "Order tester" {
		t.Fatal("cash settlement failed", err)
	}
	retry, err := f.manager.Pay(ctx, f.actor, order.ID, input)
	if err != nil || retry.ID != payment.ID {
		t.Fatal("retry duplicated payment", err)
	}
	input.RequestKey = "second_payment_12345678"
	if _, err := f.manager.Pay(ctx, f.actor, order.ID, input); !errors.Is(err, service.ErrAlreadyPaid) {
		t.Fatal("second payment accepted", err)
	}
	updated, err := f.manager.Get(ctx, order.ID)
	if err != nil || updated.Status != "RECEIVED" || updated.PaymentStatus != "PAID" || updated.Balance != 0 || updated.PaidAmount != 32000 || updated.Payment == nil {
		t.Fatal("payment changed laundry status or wrong totals", err)
	}
}
func TestConcurrentPaymentAndNonCashConfirmation(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	order, err := f.manager.Create(ctx, f.actor, f.input("noncash_order_123456789"))
	if err != nil {
		t.Fatal(err)
	}
	received := int64(32000)
	input := service.PaymentInput{Method: "QRIS", AmountReceived: &received, PaidAt: time.Now(), RequestKey: "noncash_payment_1234567"}
	if _, err := f.manager.Pay(ctx, f.actor, order.ID, input); err == nil {
		t.Fatal("unconfirmed funds accepted")
	}
	input.Confirmed = true
	var wg sync.WaitGroup
	type result struct {
		payment service.Payment
		err     error
	}
	results := make(chan result, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := f.manager.Pay(ctx, f.actor, order.ID, input)
			results <- result{p, err}
		}()
	}
	wg.Wait()
	close(results)
	var id int64
	for r := range results {
		if r.err != nil || r.payment.ChangeAmount != 0 {
			t.Fatal("concurrent settlement failed", r.err)
		}
		if id != 0 && id != r.payment.ID {
			t.Fatal("concurrent payments duplicated")
		}
		id = r.payment.ID
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.payments WHERE order_id=$1`, order.ID).Scan(&count)
	if count != 1 {
		t.Fatal("payment duplication")
	}
}

func TestPaymentStateIndependentOfLaundryState(t *testing.T) {
	for _, status := range []string{"RECEIVED", "WASHING", "COMPLETED"} {
		t.Run(status, func(t *testing.T) {
			f := setupOrders(t)
			ctx := context.Background()
			order, err := f.manager.Create(ctx, f.actor, f.input("payment_state_test_12345"))
			if err != nil {
				t.Fatal(err)
			}
			f.pool.Exec(ctx, `UPDATE laundry.orders SET status=$2 WHERE id=$1`, order.ID, status)
			received := int64(32000)
			input := service.PaymentInput{Method: "TRANSFER", AmountReceived: &received, PaidAt: time.Now(), Confirmed: true, RequestKey: "state_settlement_1234567"}
			if _, err := f.manager.Pay(ctx, f.actor, order.ID, input); err != nil {
				t.Fatal(err)
			}
			detail, err := f.manager.Get(ctx, order.ID)
			if err != nil || detail.Status != status || detail.PaidAmount != 32000 || detail.Balance != 0 || detail.Payment == nil || detail.Payment.Method != "TRANSFER" {
				t.Fatal("payment state contaminated laundry state", err)
			}
			list, err := f.manager.List(ctx, service.OrderQuery{Search: order.Number})
			if err != nil || len(list.Data) != 1 || list.Data[0].PaidAmount != 32000 || list.Data[0].Balance != 0 {
				t.Fatal("list amounts differ from detail", err)
			}
		})
	}
}

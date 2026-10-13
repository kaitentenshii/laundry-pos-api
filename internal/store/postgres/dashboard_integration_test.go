package postgres

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"testing"
	"time"
)

func TestDashboardSettledReceiptsAndBusinessDay(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	store := NewOrderStore(f.pool)
	now := time.Now()
	before, err := store.Dashboard(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	o, err := f.manager.Create(ctx, f.actor, f.input("dashboard_order_123456"))
	if err != nil {
		t.Fatal(err)
	}
	amount := int64(40000)
	_, err = f.manager.Pay(ctx, f.actor, o.ID, service.PaymentInput{Method: "CASH", AmountReceived: &amount, PaidAt: now, RequestKey: "dashboard_payment_123456"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.Dashboard(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if after.Active != before.Active+1 || after.Waiting != before.Waiting+1 || after.Receipts != before.Receipts+32000 || after.PaymentCount != before.PaymentCount+1 {
		t.Fatalf("invalid counts or cash change counted: %+v", after)
	}
	next, err := store.Dashboard(ctx, now.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if next.Receipts != 0 || next.PaymentCount != 0 {
		t.Fatal("payment leaked into next business day")
	}
}

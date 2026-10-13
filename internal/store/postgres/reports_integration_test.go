package postgres

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"testing"
	"time"
)

func TestReportDateBasisSnapshotsAndPagination(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	store := NewOrderStore(f.pool)
	var orders []service.Order
	for _, key := range []string{"report_first_order_1234", "report_last_order_12345", "report_next_order_12345"} {
		o, err := f.manager.Create(ctx, f.actor, f.input(key))
		if err != nil {
			t.Fatal(err)
		}
		orders = append(orders, o)
	}
	amount := int64(40000)
	_, err := f.manager.Pay(ctx, f.actor, orders[0].ID, service.PaymentInput{Method: "CASH", AmountReceived: &amount, PaidAt: time.Now(), RequestKey: "report_payment_key_1234"})
	if err != nil {
		t.Fatal(err)
	}
	for i, date := range []string{"2026-01-01T00:00:00+07:00", "2026-01-01T23:59:59.999999+07:00", "2026-01-02T00:00:00+07:00"} {
		if _, err = f.pool.Exec(ctx, `UPDATE laundry.orders SET created_at=$1 WHERE id=$2`, date, orders[i].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = f.pool.Exec(ctx, `UPDATE laundry.payments SET paid_at='2026-01-02T00:00:00+07:00' WHERE order_id=$1`, orders[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.pool.Exec(ctx, `UPDATE laundry.services SET price=10000 WHERE id=$1`, f.kg); err != nil {
		t.Fatal(err)
	}
	q, err := service.NormalizeReportQuery(service.ReportQuery{FromDate: "2026-01-01", ToDate: "2026-01-01", PageSize: 1}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r, err := store.Report(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != 2 || r.OrderValue != 64000 || r.Receipts != 0 || r.Balance != 32000 || len(r.Orders) != 1 || r.Orders[0].ID != orders[1].ID {
		t.Fatalf("snapshot/calendar/summary failure: %+v", r)
	}
	q.Page = 2
	r, err = store.Report(ctx, q)
	if err != nil || len(r.Orders) != 1 || r.Orders[0].ID != orders[0].ID || r.OrderValue != 64000 {
		t.Fatal("pagination changed summaries", r, err)
	}
	q, err = service.NormalizeReportQuery(service.ReportQuery{FromDate: "2026-01-02", ToDate: "2026-01-02"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r, err = store.Report(ctx, q)
	if err != nil || r.Total != 1 || r.Receipts != 32000 || r.OrderValue != 32000 || r.Balance != 32000 {
		t.Fatal("receipt basis includes wrong day or cash change", r, err)
	}
}

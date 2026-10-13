package postgres

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"testing"
)

func TestOrderStatusTransitionsAtomicAndIdempotent(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	order, err := f.manager.Create(ctx, f.actor, f.input("status_order_test_123456"))
	if err != nil {
		t.Fatal(err)
	}
	input := service.OrderStatusInput{FromStatus: "RECEIVED", Status: "WASHING", RequestKey: "washing_status_1234567"}
	updated, err := f.manager.Transition(ctx, f.actor, order.ID, input)
	if err != nil || updated.Status != "WASHING" || updated.PaymentStatus != "UNPAID" {
		t.Fatal("start washing failed", err)
	}
	replay, err := f.manager.Transition(ctx, f.actor, order.ID, input)
	if err != nil || replay.Status != "WASHING" {
		t.Fatal("retry rejected", err)
	}
	input = service.OrderStatusInput{FromStatus: "WASHING", Status: "COMPLETED", RequestKey: "complete_status_1234567"}
	updated, err = f.manager.Transition(ctx, f.actor, order.ID, input)
	if err != nil || updated.Status != "COMPLETED" {
		t.Fatal("completion failed", err)
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.order_status_history WHERE order_id=$1`, order.ID).Scan(&count)
	if count != 3 {
		t.Fatal("retry duplicated history")
	}
	var from, to, name string
	var actor int64
	f.pool.QueryRow(ctx, `SELECT from_status,to_status,actor_name,changed_by FROM laundry.order_status_history WHERE order_id=$1 ORDER BY id DESC LIMIT 1`, order.ID).Scan(&from, &to, &name, &actor)
	if from != "WASHING" || to != "COMPLETED" || actor != f.actor || name != "Order tester" {
		t.Fatal("history actor/status incorrect")
	}
}

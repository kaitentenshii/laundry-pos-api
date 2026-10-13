package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"sync"
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
	history, err := f.manager.History(ctx, order.ID)
	if err != nil || len(history) != 3 || history[0].ToStatus != "COMPLETED" || history[0].ChangedBy != f.actor || history[2].ToStatus != "RECEIVED" {
		t.Fatal("history order/actor incorrect", err)
	}
	detail, err := f.manager.Get(ctx, order.ID)
	if err != nil || detail.History[0].ToStatus != detail.Status {
		t.Fatal("history latest mismatches detail", err)
	}
}

func TestInvalidTransitionsDoNotMutateOrder(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	order, err := f.manager.Create(ctx, f.actor, f.input("invalid_status_order_12345"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := service.OrderStatusInput{FromStatus: "RECEIVED", Status: "COMPLETED", RequestKey: "invalid_jump_123456789"}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, invalid); !errors.Is(err, service.ErrInvalidTransition) {
		t.Fatal("jump accepted")
	}
	washing := service.OrderStatusInput{FromStatus: "RECEIVED", Status: "WASHING", RequestKey: "valid_wash_1234567890"}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, washing); err != nil {
		t.Fatal(err)
	}
	washing.RequestKey = "stale_state_1234567890"
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, washing); !errors.Is(err, service.ErrStaleOrder) {
		t.Fatal("stale action accepted")
	}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, service.OrderStatusInput{FromStatus: "WASHING", Status: "COMPLETED", RequestKey: "valid_complete_12345678"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, service.OrderStatusInput{FromStatus: "COMPLETED", Status: "PICKED_UP", RequestKey: "unpaid_pickup_123456789", HandoverConfirmed: true}); !errors.Is(err, service.ErrPaymentRequired) {
		t.Fatal("unpaid pickup accepted", err)
	}
	detail, err := f.manager.Get(ctx, order.ID)
	if err != nil || detail.Status != "COMPLETED" || len(detail.AllowedNextStatuses) != 0 || detail.PickupBlockedReason == "" {
		t.Fatal("invalid UI policy", err)
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.order_status_history WHERE order_id=$1`, order.ID).Scan(&count)
	if count != 3 {
		t.Fatal("invalid transition created history")
	}
}

func TestConcurrentStatusHistoryAndAtomicFailure(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	order, err := f.manager.Create(ctx, f.actor, f.input("concurrent_status_order_12345"))
	if err != nil {
		t.Fatal(err)
	}
	input := service.OrderStatusInput{FromStatus: "RECEIVED", Status: "WASHING", RequestKey: "concurrent_status_request_12345"}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := f.manager.Transition(ctx, f.actor, order.ID, input); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.order_status_history WHERE order_id=$1`, order.ID).Scan(&count)
	if count != 2 {
		t.Fatal("concurrent retry duplicated transition")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE laundry.order_status_history SET actor_name='changed' WHERE order_id=$1`, order.ID); err == nil {
		t.Fatal("history update accepted")
	}
	// Inject a database failure after the order UPDATE; the transaction must roll it back.
	name := fmt.Sprintf("qa_status_failure_%d", order.ID)
	functionSQL := fmt.Sprintf(`CREATE FUNCTION laundry.%s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.order_id=%d AND NEW.to_status='COMPLETED' THEN RAISE EXCEPTION 'injected QA failure'; END IF; RETURN NEW; END $$`, name, order.ID)
	if _, err := f.pool.Exec(ctx, functionSQL); err != nil {
		t.Fatal(err)
	}
	defer f.pool.Exec(ctx, fmt.Sprintf(`DROP FUNCTION laundry.%s() CASCADE`, name))
	if _, err := f.pool.Exec(ctx, fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON laundry.order_status_history FOR EACH ROW EXECUTE FUNCTION laundry.%s()`, name, name)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, service.OrderStatusInput{FromStatus: "WASHING", Status: "COMPLETED", RequestKey: "failed_history_insert_12345"}); err == nil {
		t.Fatal("injected failure ignored")
	}
	current, err := f.manager.Get(ctx, order.ID)
	if err != nil || current.Status != "WASHING" {
		t.Fatal("order update escaped rollback", err)
	}
	f.pool.QueryRow(ctx, `SELECT count(*) FROM laundry.order_status_history WHERE order_id=$1`, order.ID).Scan(&count)
	if count != 2 {
		t.Fatal("failed history insertion persisted")
	}
}

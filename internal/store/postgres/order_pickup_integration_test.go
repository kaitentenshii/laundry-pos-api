package postgres

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"testing"
	"time"
)

func TestCompletedPaidOrderPickup(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	order, err := f.manager.Create(ctx, f.actor, f.input("pickup_order_test_123456"))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []service.OrderStatusInput{{FromStatus: "RECEIVED", Status: "WASHING", RequestKey: "pickup_start_wash_123456"}, {FromStatus: "WASHING", Status: "COMPLETED", RequestKey: "pickup_complete_1234567"}} {
		if _, err := f.manager.Transition(ctx, f.actor, order.ID, input); err != nil {
			t.Fatal(err)
		}
	}
	pickup := service.OrderStatusInput{FromStatus: "COMPLETED", Status: "PICKED_UP", RequestKey: "pickup_confirm_12345678"}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, pickup); err == nil {
		t.Fatal("unconfirmed handover accepted")
	}
	amount := int64(32000)
	payment, err := f.manager.Pay(ctx, f.actor, order.ID, service.PaymentInput{Method: "CASH", AmountReceived: &amount, PaidAt: time.Now(), RequestKey: "pickup_pay_12345678901"})
	if err != nil {
		t.Fatal(err)
	}
	pickup.HandoverConfirmed = true
	result, err := f.manager.Transition(ctx, f.actor, order.ID, pickup)
	if err != nil || result.Status != "PICKED_UP" || result.Payment == nil || result.Payment.ID != payment.ID || !result.History[0].HandoverConfirmed || len(result.AllowedNextStatuses) != 0 {
		t.Fatal("pickup lost payment or confirmation", err)
	}
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, pickup); err != nil {
		t.Fatal("identical pickup retry failed", err)
	}
	pickup.RequestKey = "second_pickup_123456789"
	if _, err := f.manager.Transition(ctx, f.actor, order.ID, pickup); err == nil {
		t.Fatal("second handover accepted")
	}
	history, err := f.manager.History(ctx, order.ID)
	if err != nil || len(history) != 4 {
		t.Fatal("pickup history duplicated", err)
	}
}

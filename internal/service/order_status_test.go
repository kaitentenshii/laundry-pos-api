package service

import (
	"errors"
	"testing"
)

func TestLaundryTransitionMatrix(t *testing.T) {
	statuses := []string{"RECEIVED", "WASHING", "COMPLETED", "PICKED_UP"}
	for i, from := range statuses {
		for j, to := range statuses {
			for _, payment := range []string{"UNPAID", "PAID"} {
				err := ValidateOrderTransition(from, to, payment)
				allowed := j == i+1 && (to != "PICKED_UP" || payment == "PAID")
				if (err == nil) != allowed {
					t.Fatalf("%s -> %s payment=%s err=%v", from, to, payment, err)
				}
			}
		}
	}
	if !errors.Is(ValidateOrderTransition("COMPLETED", "PICKED_UP", "UNPAID"), ErrPaymentRequired) {
		t.Fatal("missing payment guard")
	}
	if len(AllowedOrderTransitions("COMPLETED", "UNPAID")) != 0 || len(AllowedOrderTransitions("PICKED_UP", "PAID")) != 0 {
		t.Fatal("terminal/unpaid transition exposed")
	}
}

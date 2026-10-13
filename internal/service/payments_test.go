package service

import (
	"context"
	"testing"
	"time"
)

func TestInvalidPaymentInputs(t *testing.T) {
	amount := int64(32000)
	base := PaymentInput{Method: "CASH", AmountReceived: &amount, PaidAt: time.Now(), RequestKey: "payment_validation_1234"}
	manager := NewOrders(nil)
	for _, change := range []func(*PaymentInput){func(p *PaymentInput) { p.Method = "CARD" }, func(p *PaymentInput) { p.AmountReceived = nil }, func(p *PaymentInput) { p.Method = "TRANSFER"; p.Confirmed = false }, func(p *PaymentInput) { p.PaidAt = time.Now().Add(time.Hour) }, func(p *PaymentInput) { p.RequestKey = "short" }} {
		input := base
		change(&input)
		if _, err := manager.Pay(context.Background(), 1, 1, input); err == nil {
			t.Fatal("invalid payment accepted")
		}
	}
}

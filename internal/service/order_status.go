package service

import (
	"context"
	"errors"
)

var ErrInvalidTransition = errors.New("Perubahan status harus mengikuti Diterima → Dicuci → Selesai → Diambil.")
var ErrPaymentRequired = errors.New("Pesanan harus selesai dan lunas sebelum diambil.")
var ErrStaleOrder = errors.New("Status pesanan sudah berubah. Muat ulang detail pesanan.")

type OrderStatusInput struct {
	FromStatus string `json:"from_status"`
	Status     string `json:"status"`
	RequestKey string `json:"request_key"`
}

func ValidateOrderTransition(from, to, payment string) error {
	next := map[string]string{"RECEIVED": "WASHING", "WASHING": "COMPLETED", "COMPLETED": "PICKED_UP"}
	if next[from] != to || to == "" {
		return ErrInvalidTransition
	}
	if to == "PICKED_UP" && payment != "PAID" {
		return ErrPaymentRequired
	}
	return nil
}

func AllowedOrderTransitions(status, payment string) []string {
	next := map[string]string{"RECEIVED": "WASHING", "WASHING": "COMPLETED", "COMPLETED": "PICKED_UP"}[status]
	if next == "" || ValidateOrderTransition(status, next, payment) != nil {
		return []string{}
	}
	return []string{next}
}
func (s *Orders) Transition(ctx context.Context, actor, id int64, input OrderStatusInput) (Order, error) {
	if !requestKeyPattern.MatchString(input.RequestKey) {
		return Order{}, Invalid("request_key", "Kunci permintaan tidak valid.")
	}
	if err := ValidateOrderTransition(input.FromStatus, input.Status, "PAID"); err != nil {
		return Order{}, err
	}
	return s.store.TransitionOrder(ctx, actor, id, input)
}

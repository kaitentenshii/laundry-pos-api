package service

import (
	"context"
	"time"
)

type Dashboard struct {
	AsOf         time.Time `json:"as_of"`
	Active       int64     `json:"active"`
	Washing      int64     `json:"washing"`
	Waiting      int64     `json:"waiting"`
	Ready        int64     `json:"ready"`
	Overdue      int64     `json:"overdue"`
	Receipts     int64     `json:"receipts"`
	PaymentCount int64     `json:"payment_count"`
	BusinessDate string    `json:"business_date"`
	Timezone     string    `json:"timezone"`
	Monitored    []Order   `json:"monitored"`
}

type DashboardStore interface {
	Dashboard(context.Context, time.Time) (Dashboard, error)
}
type DashboardService struct{ store DashboardStore }

func NewDashboard(store DashboardStore) *DashboardService { return &DashboardService{store} }
func (s *DashboardService) Get(ctx context.Context) (Dashboard, error) {
	return s.store.Dashboard(ctx, time.Now())
}

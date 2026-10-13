package service

import (
	"context"
	"time"
)

type OrderQuery struct {
	Status, PaymentStatus, FromDate, ToDate string
	Page, PageSize                          int
	From, To                                *time.Time
}
type OrderList struct {
	Data     []Order `json:"data"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

func (s *Orders) List(ctx context.Context, q OrderQuery) (OrderList, error) {
	if q.Status != "" && q.Status != "RECEIVED" && q.Status != "WASHING" && q.Status != "COMPLETED" && q.Status != "PICKED_UP" {
		return OrderList{}, Invalid("status", "Status cucian tidak valid.")
	}
	if q.PaymentStatus != "" && q.PaymentStatus != "PAID" && q.PaymentStatus != "UNPAID" {
		return OrderList{}, Invalid("payment_status", "Status pembayaran tidak valid.")
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || q.Page > 1000000 || q.PageSize < 1 || q.PageSize > 100 {
		return OrderList{}, Invalid("page", "Pagination tidak valid.")
	}
	zone := time.FixedZone("Asia/Jakarta", 7*60*60)
	for _, entry := range []struct {
		value  string
		target **time.Time
		field  string
	}{{q.FromDate, &q.From, "from"}, {q.ToDate, &q.To, "to"}} {
		if entry.value != "" {
			parsed, err := time.ParseInLocation("2006-01-02", entry.value, zone)
			if err != nil {
				return OrderList{}, Invalid(entry.field, "Tanggal tidak valid.")
			}
			*entry.target = &parsed
		}
	}
	if q.From != nil && q.To != nil && q.From.After(*q.To) {
		return OrderList{}, Invalid("to", "Tanggal akhir tidak boleh sebelum tanggal awal.")
	}
	if q.To != nil {
		end := q.To.AddDate(0, 0, 1)
		q.To = &end
	}
	return s.store.ListOrders(ctx, q)
}

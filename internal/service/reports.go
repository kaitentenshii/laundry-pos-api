package service

import (
	"context"
	"time"
)

type ReportQuery struct {
	FromDate, ToDate string
	Page, PageSize   int
	From, To         time.Time
}
type TransactionReport struct {
	Orders     []Order `json:"orders"`
	Total      int64   `json:"total"`
	OrderValue int64   `json:"order_value"`
	Receipts   int64   `json:"receipts"`
	Balance    int64   `json:"balance"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	From       string  `json:"from"`
	To         string  `json:"to"`
	Timezone   string  `json:"timezone"`
}
type ReportStore interface {
	Report(context.Context, ReportQuery) (TransactionReport, error)
}
type Reports struct{ store ReportStore }

func NewReports(store ReportStore) *Reports { return &Reports{store} }
func NormalizeReportQuery(q ReportQuery, now time.Time) (ReportQuery, error) {
	zone := time.FixedZone("Asia/Jakarta", 7*60*60)
	local := now.In(zone)
	if q.FromDate == "" && q.ToDate == "" {
		q.FromDate = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, zone).Format("2006-01-02")
		q.ToDate = local.Format("2006-01-02")
	}
	start, err := time.ParseInLocation("2006-01-02", q.FromDate, zone)
	if err != nil {
		return q, Invalid("from", "Tanggal mulai wajib valid.")
	}
	end, err := time.ParseInLocation("2006-01-02", q.ToDate, zone)
	if err != nil {
		return q, Invalid("to", "Tanggal akhir wajib valid.")
	}
	if start.After(end) {
		return q, Invalid("to", "Tanggal akhir tidak boleh sebelum tanggal mulai.")
	}
	q.From = start
	q.To = end.AddDate(0, 0, 1)
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.Page < 1 || q.Page > 1000000 || q.PageSize < 1 || q.PageSize > 100 {
		return q, Invalid("page", "Pagination tidak valid.")
	}
	return q, nil
}
func (s *Reports) Get(ctx context.Context, q ReportQuery) (TransactionReport, error) {
	q, err := NormalizeReportQuery(q, time.Now())
	if err != nil {
		return TransactionReport{}, err
	}
	return s.store.Report(ctx, q)
}

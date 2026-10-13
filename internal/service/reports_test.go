package service

import (
	"testing"
	"time"
)

func TestReportCalendarBoundary(t *testing.T) {
	q, err := NormalizeReportQuery(ReportQuery{FromDate: "2026-10-04", ToDate: "2026-10-04"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if q.From.UTC().Format(time.RFC3339) != "2026-10-03T17:00:00Z" || q.To.UTC().Format(time.RFC3339) != "2026-10-04T17:00:00Z" {
		t.Fatal("inclusive WIB calendar range wrong", q)
	}
	for _, bad := range []ReportQuery{{FromDate: "2026-02-30", ToDate: "2026-03-01"}, {FromDate: "2026-10-05", ToDate: "2026-10-04"}, {FromDate: "2026-10-04"}, {Page: -1}, {PageSize: 101}} {
		if _, err := NormalizeReportQuery(bad, time.Now()); err == nil {
			t.Fatal("invalid query accepted", bad)
		}
	}
	q, err = NormalizeReportQuery(ReportQuery{}, time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC))
	if err != nil || q.FromDate != "2026-11-01" || q.ToDate != "2026-11-01" {
		t.Fatal("default range must use business timezone", q, err)
	}
}

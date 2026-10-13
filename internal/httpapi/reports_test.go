package httpapi

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"io"
	"log/slog"
	"testing"
)

type reportStub struct{}

func (reportStub) Get(context.Context, service.ReportQuery) (service.TransactionReport, error) {
	return service.TransactionReport{}, nil
}
func TestReportAdminOnly(t *testing.T) {
	for _, role := range []string{"admin", "cashier"} {
		router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Reports: reportStub{}})
		want := 200
		if role == "cashier" {
			want = 403
		}
		if r := authRequest(router, "GET", "/reports/transactions", "", "valid-token"); r.Code != want {
			t.Fatalf("%s status %d", role, r.Code)
		}
		if r := authRequest(router, "GET", "/reports/transactions", "", ""); r.Code != 401 {
			t.Fatal("anonymous report accepted")
		}
		if role == "admin" {
			if r := authRequest(router, "GET", "/reports/transactions?page=invalid", "", "valid-token"); r.Code != 400 {
				t.Fatal("invalid page accepted")
			}
		}
	}
}

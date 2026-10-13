package httpapi

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"io"
	"log/slog"
	"testing"
)

type dashboardStub struct{}

func (dashboardStub) Get(context.Context) (service.Dashboard, error) {
	return service.Dashboard{Active: 3}, nil
}
func TestDashboardAuthentication(t *testing.T) {
	for _, role := range []string{"admin", "cashier"} {
		router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Dashboard: dashboardStub{}})
		if r := authRequest(router, "GET", "/dashboard", "", "valid-token"); r.Code != 200 {
			t.Fatalf("%s denied: %d", role, r.Code)
		}
		if r := authRequest(router, "GET", "/dashboard", "", ""); r.Code != 401 {
			t.Fatal("unauthenticated dashboard allowed")
		}
	}
}

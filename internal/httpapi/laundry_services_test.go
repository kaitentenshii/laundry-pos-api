package httpapi

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"io"
	"log/slog"
	"testing"
)

type laundryServiceStub struct{ activeOnly bool }

func (s *laundryServiceStub) List(_ context.Context, active bool) ([]service.LaundryService, error) {
	s.activeOnly = active
	return []service.LaundryService{}, nil
}
func (*laundryServiceStub) Create(_ context.Context, input service.LaundryServiceInput) (service.LaundryService, error) {
	return service.LaundryService{ID: 1, Name: input.Name}, nil
}
func (*laundryServiceStub) Update(_ context.Context, id int64, input service.LaundryServiceInput) (service.LaundryService, error) {
	return service.LaundryService{ID: id, Name: input.Name}, nil
}
func TestLaundryServiceRoles(t *testing.T) {
	for _, role := range []string{"admin", "cashier"} {
		manager := &laundryServiceStub{}
		router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Services: manager})
		if authRequest(router, "GET", "/services?active=false", "", "valid-token").Code != 200 || manager.activeOnly != (role == "cashier") {
			t.Fatal("cashier bypassed active filter")
		}
		for _, method := range []string{"POST", "PUT"} {
			path := "/services"
			if method == "PUT" {
				path += "/1"
			}
			status := authRequest(router, method, path, `{"name":"Cuci","unit":"KG","price":8000,"estimated_hours":48,"is_active":true}`, "valid-token").Code
			if role == "cashier" && status != 403 || role == "admin" && (status < 200 || status >= 300) {
				t.Fatalf("invalid %s authorization: %d", role, status)
			}
			if authRequest(router, method, path, "{}", "").Code != 401 {
				t.Fatal("unauthenticated service mutation")
			}
		}
	}
}

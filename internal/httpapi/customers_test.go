package httpapi

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"io"
	"log/slog"
	"testing"
)

type customerServiceStub struct{}

func (customerServiceStub) List(context.Context) ([]service.Customer, error) {
	return []service.Customer{}, nil
}
func (customerServiceStub) Create(_ context.Context, input service.CustomerInput) (service.Customer, error) {
	return service.Customer{ID: 1, CustomerInput: input}, nil
}
func TestCustomerRoutesRoles(t *testing.T) {
	for _, role := range []string{"admin", "cashier", "unknown"} {
		router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Customers: customerServiceStub{}})
		for _, method := range []string{"GET", "POST"} {
			result := authRequest(router, method, "/customers", `{"name":"Sari","whatsapp":"081234567890"}`, "valid-token")
			if role == "unknown" {
				if result.Code != 403 {
					t.Fatal("unknown role accepted")
				}
			} else if result.Code < 200 || result.Code >= 300 {
				t.Fatalf("%s %s: %d", role, method, result.Code)
			}
			if authRequest(router, method, "/customers", "{}", "").Code != 401 {
				t.Fatal("unauthenticated customer access")
			}
		}
		if role == "cashier" && authRequest(router, "POST", "/customers", `{"unexpected":1}`, "valid-token").Code != 400 {
			t.Fatal("unknown payload field accepted")
		}
	}
}

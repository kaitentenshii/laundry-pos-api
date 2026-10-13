package httpapi

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"io"
	"log/slog"
	"testing"
)

type orderManagerStub struct{}

func (orderManagerStub) History(context.Context, int64) ([]service.OrderStatusHistory, error) {
	return []service.OrderStatusHistory{}, nil
}

func (orderManagerStub) Transition(context.Context, int64, int64, service.OrderStatusInput) (service.Order, error) {
	return service.Order{ID: 1, Status: "WASHING"}, nil
}

func (orderManagerStub) Pay(context.Context, int64, int64, service.PaymentInput) (service.Payment, error) {
	return service.Payment{ID: 1}, nil
}

func (orderManagerStub) List(context.Context, service.OrderQuery) (service.OrderList, error) {
	return service.OrderList{Data: []service.Order{}}, nil
}

func (orderManagerStub) Create(_ context.Context, actor int64, input service.OrderInput) (service.Order, error) {
	return service.Order{ID: 1, CreatedBy: actor, CustomerID: input.CustomerID, Status: "RECEIVED", PaymentStatus: "UNPAID"}, nil
}
func (orderManagerStub) Get(_ context.Context, id int64) (service.Order, error) {
	if id == 999 {
		return service.Order{}, service.ErrNotFound
	}
	return service.Order{ID: id}, nil
}
func TestOrderRouteAuthorization(t *testing.T) {
	for _, role := range []string{"admin", "cashier", "unknown"} {
		router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Orders: orderManagerStub{}})
		for _, endpoint := range []struct{ method, path string }{{"POST", "/orders"}, {"GET", "/orders/1"}} {
			response := authRequest(router, endpoint.method, endpoint.path, `{"customer_id":1}`, "valid-token")
			if role == "unknown" {
				if response.Code != 403 {
					t.Fatal("unknown role accepted")
				}
			} else if response.Code < 200 || response.Code >= 300 {
				t.Fatal("operational role denied")
			}
			if authRequest(router, endpoint.method, endpoint.path, "{}", "").Code != 401 {
				t.Fatal("unauthenticated order access")
			}
		}
		if role == "cashier" && authRequest(router, "GET", "/orders/999", "", "valid-token").Code != 404 {
			t.Fatal("order not-found status invalid")
		}
	}
}

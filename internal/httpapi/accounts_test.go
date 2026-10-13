package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type accountServiceStub struct{}

func (accountServiceStub) List(context.Context) ([]service.Account, error) {
	return []service.Account{{ID: 1, Name: "Admin", Role: "admin", IsActive: true}}, nil
}
func (accountServiceStub) Create(context.Context, service.CreateAccountInput) (service.Account, error) {
	return service.Account{ID: 2, Role: "cashier"}, nil
}
func (accountServiceStub) Update(context.Context, int64, int64, service.AccountInput) (service.Account, error) {
	return service.Account{ID: 2, Role: "cashier"}, nil
}

type authRoleStub struct {
	authStub
	role string
}

func (s *authRoleStub) Authenticate(ctx context.Context, token string) (service.User, error) {
	u, err := s.authStub.Authenticate(ctx, token)
	u.Role = s.role
	return u, err
}

func TestUserRoutesEnforceAdmin(t *testing.T) {
	for _, role := range []string{"admin", "cashier"} {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		router := NewRouter(logger, readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Accounts: accountServiceStub{}})
		for _, endpoint := range []struct{ method, path, body string }{{"GET", "/users", ""}, {"POST", "/users", `{"name":"Kasir","identifier":"cashier","role":"cashier","password":"password1234"}`}, {"PUT", "/users/2", `{"name":"Kasir","identifier":"cashier","role":"cashier","is_active":true}`}} {
			r := authRequest(router, endpoint.method, endpoint.path, endpoint.body, "valid-token")
			if role == "cashier" && r.Code != 403 {
				t.Fatalf("cashier %s allowed", endpoint.method)
			}
			if role == "admin" && (r.Code < 200 || r.Code >= 300) {
				t.Fatalf("admin %s denied: %d", endpoint.method, r.Code)
			}
			if strings.Contains(r.Body.String(), "password_hash") {
				t.Fatal("response leaked password hash")
			}
			if noAuth := authRequest(router, endpoint.method, endpoint.path, endpoint.body, ""); noAuth.Code != 401 {
				t.Fatal("user endpoint allowed unauthenticated access")
			}
		}
		if role == "admin" {
			request := httptest.NewRequest("POST", "/users", strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != 401 {
				t.Fatal("authentication must run before payload handling")
			}
		}
	}
}

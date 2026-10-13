package httpapi

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"io"
	"log/slog"
	"testing"
)

type profileStub struct{}

func (profileStub) Get(context.Context) (service.LaundryProfile, error) {
	return service.LaundryProfile{ProfileInput: service.ProfileInput{Name: "Laundry Mitra", Contact: "private"}}, nil
}
func (profileStub) Save(context.Context, service.ProfileInput) (service.LaundryProfile, error) {
	return service.LaundryProfile{}, nil
}
func TestProfileRoleBoundaries(t *testing.T) {
	for _, role := range []string{"admin", "cashier"} {
		router := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: &authRoleStub{role: role}, Profile: profileStub{}})
		for _, method := range []string{"GET", "PUT"} {
			want := 200
			if role == "cashier" {
				want = 403
			}
			r := authRequest(router, method, "/settings/profile", "{}", "valid-token")
			if r.Code != want {
				t.Fatal("settings role failure", role, method, r.Code)
			}
		}
		r := authRequest(router, "GET", "/laundry/identity", "", "valid-token")
		if r.Code != 200 {
			t.Fatal("identity denied")
		}
		if r := authRequest(router, "GET", "/settings/profile", "", ""); r.Code != 401 {
			t.Fatal("anonymous profile accepted")
		}
	}
}

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

func TestAdminRoleAuthorization(t *testing.T) {
	for _, test := range []struct {
		role          string
		authenticated bool
		want          int
	}{
		{"admin", true, 204}, {"cashier", true, 403}, {"unknown", true, 403}, {"", false, 401},
	} {
		request := httptest.NewRequest("GET", "/admin", nil)
		if test.authenticated {
			request = request.WithContext(context.WithValue(request.Context(), userContextKey{}, service.User{Role: test.role}))
		}
		recorder := httptest.NewRecorder()
		requireRoles("admin")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(recorder, request)
		if recorder.Code != test.want {
			t.Fatalf("role=%s status=%d want=%d", test.role, recorder.Code, test.want)
		}
	}
}

func TestRolePermissionMatrix(t *testing.T) {
	contains := func(values []string, target string) bool {
		for _, value := range values {
			if value == target {
				return true
			}
		}
		return false
	}
	for _, permission := range []string{"users:manage", "services:manage", "settings:manage", "reports:read"} {
		if !contains(rolePermissions("admin"), permission) || contains(rolePermissions("cashier"), permission) {
			t.Fatalf("invalid admin permission %s", permission)
		}
	}
	for _, permission := range []string{"customers:manage", "orders:manage", "notifications:read", "notifications:retry"} {
		if !contains(rolePermissions("admin"), permission) || !contains(rolePermissions("cashier"), permission) {
			t.Fatalf("missing operational permission %s", permission)
		}
	}
	if len(rolePermissions("unknown")) != 0 {
		t.Fatal("unknown roles must not have permissions")
	}
}

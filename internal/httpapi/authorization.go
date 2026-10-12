package httpapi

import (
	"net/http"

	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

func requestUser(r *http.Request) (service.User, bool) {
	user, ok := r.Context().Value(userContextKey{}).(service.User)
	return user, ok
}

func requireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := requestUser(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
				return
			}
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeError(w, http.StatusForbidden, "forbidden", "Your role cannot access this resource.")
		})
	}
}

func rolePermissions(role string) []string {
	permissions := []string{"dashboard:read", "customers:manage", "orders:manage", "payments:record", "notifications:read", "notifications:retry"}
	if role == "admin" {
		return append(permissions, "users:manage", "services:manage", "settings:manage", "reports:read")
	}
	if role == "cashier" {
		return permissions
	}
	return []string{}
}

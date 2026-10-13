package httpapi

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"log/slog"
	"net/http"
	"time"
)

type CustomerService interface {
	List(context.Context) ([]service.Customer, error)
	Create(context.Context, service.CustomerInput) (service.Customer, error)
}

func registerCustomerRoutes(router chi.Router, auth AuthService, customers CustomerService, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin", "cashier"))
		r.Get("/customers", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := customers.List(ctx)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Post("/customers", func(w http.ResponseWriter, r *http.Request) {
			var input service.CustomerInput
			if !decodeInput(w, r, &input) {
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := customers.Create(ctx, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 201, map[string]any{"data": data})
		})
	})
}

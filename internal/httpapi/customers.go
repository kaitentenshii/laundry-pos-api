package httpapi

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type CustomerService interface {
	Get(context.Context, int64) (service.Customer, error)
	List(context.Context, service.CustomerQuery) (service.CustomerList, error)
	Create(context.Context, service.CustomerInput) (service.Customer, error)
	Update(context.Context, int64, service.CustomerInput) (service.Customer, error)
}

func registerCustomerRoutes(router chi.Router, auth AuthService, customers CustomerService, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin", "cashier"))
		r.Get("/customers/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, ok := resourceID(w, r)
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := customers.Get(ctx, id)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Put("/customers/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, ok := resourceID(w, r)
			if !ok {
				return
			}
			var input service.CustomerInput
			if !decodeInput(w, r, &input) {
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := customers.Update(ctx, id, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Get("/customers", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			query := service.CustomerQuery{Search: r.URL.Query().Get("q")}
			for key, target := range map[string]*int{"page": &query.Page, "page_size": &query.PageSize} {
				if value := r.URL.Query().Get(key); value != "" {
					parsed, err := strconv.Atoi(value)
					if err != nil || parsed < 1 {
						managementError(w, logger, service.Invalid(key, "Pagination tidak valid."))
						return
					}
					*target = parsed
				}
			}
			if value := r.URL.Query().Get("consent"); value != "" {
				if value != "true" && value != "false" {
					managementError(w, logger, service.Invalid("consent", "Filter persetujuan tidak valid."))
					return
				}
				consent := value == "true"
				query.Consent = &consent
			}
			data, err := customers.List(ctx, query)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, data)
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

package httpapi

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"log/slog"
	"net/http"
	"time"
)

type LaundryServiceManager interface {
	Prices(context.Context, int64) ([]service.ServicePriceVersion, error)
	List(context.Context, bool) ([]service.LaundryService, error)
	Create(context.Context, service.LaundryServiceInput) (service.LaundryService, error)
	Update(context.Context, int64, service.LaundryServiceInput) (service.LaundryService, error)
}

func registerLaundryServiceRoutes(router chi.Router, auth AuthService, manager LaundryServiceManager, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin", "cashier"))
		r.Get("/services", func(w http.ResponseWriter, r *http.Request) {
			user, _ := requestUser(r)
			activeOnly := user.Role == "cashier"
			if filter := r.URL.Query().Get("active"); filter != "" {
				if filter != "true" && filter != "false" {
					managementError(w, logger, service.Invalid("active", "Filter status tidak valid."))
					return
				}
				activeOnly = activeOnly || filter == "true"
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := manager.List(ctx, activeOnly)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Group(func(r chi.Router) {
			r.Use(requireRoles("admin"))
			r.Get("/services/{id}/prices", func(w http.ResponseWriter, r *http.Request) {
				id, ok := resourceID(w, r)
				if !ok {
					return
				}
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				data, err := manager.Prices(ctx, id)
				if err != nil {
					managementError(w, logger, err)
					return
				}
				writeJSON(w, 200, map[string]any{"data": data})
			})
			r.Post("/services", func(w http.ResponseWriter, r *http.Request) {
				var input service.LaundryServiceInput
				if !decodeInput(w, r, &input) {
					return
				}
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				data, err := manager.Create(ctx, input)
				if err != nil {
					managementError(w, logger, err)
					return
				}
				writeJSON(w, 201, map[string]any{"data": data})
			})
			r.Put("/services/{id}", func(w http.ResponseWriter, r *http.Request) {
				id, ok := resourceID(w, r)
				if !ok {
					return
				}
				var input service.LaundryServiceInput
				if !decodeInput(w, r, &input) {
					return
				}
				ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				defer cancel()
				data, err := manager.Update(ctx, id, input)
				if err != nil {
					managementError(w, logger, err)
					return
				}
				writeJSON(w, 200, map[string]any{"data": data})
			})
		})
	})
}

package httpapi

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"log/slog"
	"net/http"
	"time"
)

type ProfileManager interface {
	Get(context.Context) (service.LaundryProfile, error)
	Save(context.Context, service.ProfileInput) (service.LaundryProfile, error)
}

func registerProfileRoutes(router chi.Router, auth AuthService, manager ProfileManager, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin", "cashier"))
		r.Get("/laundry/identity", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := manager.Get(ctx)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": map[string]string{"name": data.Name}})
		})
	})
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin"))
		r.Get("/settings/profile", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := manager.Get(ctx)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Put("/settings/profile", func(w http.ResponseWriter, r *http.Request) {
			var input service.ProfileInput
			if !decodeInput(w, r, &input) {
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := manager.Save(ctx, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
	})
}

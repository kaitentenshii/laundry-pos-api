package httpapi

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"log/slog"
	"net/http"
	"time"
)

type DashboardManager interface {
	Get(context.Context) (service.Dashboard, error)
}

func registerDashboardRoutes(router chi.Router, auth AuthService, manager DashboardManager, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin", "cashier"))
		r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := manager.Get(ctx)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
	})
}

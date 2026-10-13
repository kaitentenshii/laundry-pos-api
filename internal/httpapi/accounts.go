package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type AccountService interface {
	List(context.Context) ([]service.Account, error)
	Create(context.Context, service.CreateAccountInput) (service.Account, error)
	Update(context.Context, int64, int64, service.AccountInput) (service.Account, error)
}

func registerAccountRoutes(router chi.Router, auth AuthService, accounts AccountService, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin"))
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := accounts.List(ctx)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Post("/users", func(w http.ResponseWriter, r *http.Request) {
			var input service.CreateAccountInput
			if !decodeInput(w, r, &input) {
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			a, err := accounts.Create(ctx, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 201, map[string]any{"data": a})
		})
		r.Put("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, ok := resourceID(w, r)
			if !ok {
				return
			}
			var input service.AccountInput
			if !decodeInput(w, r, &input) {
				return
			}
			user, _ := requestUser(r)
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			a, err := accounts.Update(ctx, user.ID, id, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": a})
		})
	})
}

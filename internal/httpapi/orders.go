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

type OrderManager interface {
	Transition(context.Context, int64, int64, service.OrderStatusInput) (service.Order, error)
	Pay(context.Context, int64, int64, service.PaymentInput) (service.Payment, error)
	List(context.Context, service.OrderQuery) (service.OrderList, error)
	Create(context.Context, int64, service.OrderInput) (service.Order, error)
	Get(context.Context, int64) (service.Order, error)
}

func registerOrderRoutes(router chi.Router, auth AuthService, orders OrderManager, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin", "cashier"))
		r.Patch("/orders/{id}/status", func(w http.ResponseWriter, r *http.Request) {
			id, ok := resourceID(w, r)
			if !ok {
				return
			}
			var input service.OrderStatusInput
			if !decodeInput(w, r, &input) {
				return
			}
			actor, _ := requestUser(r)
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := orders.Transition(ctx, actor.ID, id, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
		r.Post("/orders/{id}/payments", func(w http.ResponseWriter, r *http.Request) {
			id, ok := resourceID(w, r)
			if !ok {
				return
			}
			var input service.PaymentInput
			if !decodeInput(w, r, &input) {
				return
			}
			actor, _ := requestUser(r)
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := orders.Pay(ctx, actor.ID, id, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 201, map[string]any{"data": data})
		})
		r.Get("/orders", func(w http.ResponseWriter, r *http.Request) {
			values := r.URL.Query()
			q := service.OrderQuery{Search: values.Get("q"), Status: values.Get("status"), PaymentStatus: values.Get("payment_status"), FromDate: values.Get("from"), ToDate: values.Get("to")}
			for key, target := range map[string]*int{"page": &q.Page, "page_size": &q.PageSize} {
				if value := values.Get(key); value != "" {
					parsed, err := strconv.Atoi(value)
					if err != nil || parsed < 1 {
						managementError(w, logger, service.Invalid(key, "Pagination tidak valid."))
						return
					}
					*target = parsed
				}
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := orders.List(ctx, q)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, data)
		})
		r.Post("/orders", func(w http.ResponseWriter, r *http.Request) {
			var input service.OrderInput
			if !decodeInput(w, r, &input) {
				return
			}
			actor, _ := requestUser(r)
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := orders.Create(ctx, actor.ID, input)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 201, map[string]any{"data": data})
		})
		r.Get("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, ok := resourceID(w, r)
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := orders.Get(ctx, id)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
	})
}

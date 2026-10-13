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

type ReportManager interface {
	Get(context.Context, service.ReportQuery) (service.TransactionReport, error)
}

func registerReportRoutes(router chi.Router, auth AuthService, manager ReportManager, logger *slog.Logger) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth(auth, logger))
		r.Use(requireRoles("admin"))
		r.Get("/reports/transactions", func(w http.ResponseWriter, r *http.Request) {
			v := r.URL.Query()
			q := service.ReportQuery{FromDate: v.Get("from"), ToDate: v.Get("to")}
			for name, target := range map[string]*int{"page": &q.Page, "page_size": &q.PageSize} {
				if value := v.Get(name); value != "" {
					parsed, e := strconv.Atoi(value)
					if e != nil || parsed < 1 {
						managementError(w, logger, service.Invalid(name, "Pagination tidak valid."))
						return
					}
					*target = parsed
				}
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			data, err := manager.Get(ctx, q)
			if err != nil {
				managementError(w, logger, err)
				return
			}
			writeJSON(w, 200, map[string]any{"data": data})
		})
	})
}

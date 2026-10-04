package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(logger *slog.Logger, readinessChecker ReadinessChecker) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(recoverer(logger))

	router.Get("/health", health)
	router.Get("/ready", readiness(readinessChecker))

	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "The requested resource was not found.")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "The request method is not allowed for this resource.")
	})

	return router
}

func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "error", recovered, "request_id", middleware.GetReqID(r.Context()))
					writeError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

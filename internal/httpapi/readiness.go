package httpapi

import (
	"context"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

type ReadinessChecker interface {
	Ping(context.Context) error
}

func readiness(checker ReadinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if err := checker.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database_unavailable", "The database is unavailable.")
			return
		}

		writeJSON(w, http.StatusOK, healthResponse{Status: "ready"})
	}
}

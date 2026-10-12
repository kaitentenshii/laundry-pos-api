package httpapi

import (
	"net/http"
	"strings"
)

func cors(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			if origin != "" {
				if !allowed[origin] {
					writeError(w, http.StatusForbidden, "origin_not_allowed", "This origin is not allowed.")
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Add("Vary", "Access-Control-Request-Method")
					w.Header().Add("Vary", "Access-Control-Request-Headers")
					requestedMethod := r.Header.Get("Access-Control-Request-Method")
					if requestedMethod != "GET" && requestedMethod != "POST" && requestedMethod != "PUT" && requestedMethod != "PATCH" {
						writeError(w, http.StatusForbidden, "method_not_allowed", "This method is not allowed.")
						return
					}
					for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
						header = strings.ToLower(strings.TrimSpace(header))
						if header != "" && header != "authorization" && header != "content-type" {
							writeError(w, http.StatusForbidden, "header_not_allowed", "This header is not allowed.")
							return
						}
					}
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

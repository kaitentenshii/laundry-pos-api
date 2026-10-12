package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type AuthService interface {
	Login(context.Context, string, string) (service.LoginResult, error)
	Authenticate(context.Context, string) (service.User, error)
	Logout(context.Context, string) error
}

type userContextKey struct{}

func registerAuthRoutes(router chi.Router, auth AuthService, logger *slog.Logger) {
	router.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Use application/json.")
			return
		}
		var input struct {
			Identifier string `json:"identifier"`
			Password   string `json:"password"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Provide a valid login request.")
			return
		}
		if decoder.Decode(&struct{}{}) != io.EOF || strings.TrimSpace(input.Identifier) == "" || len(input.Identifier) > 320 || input.Password == "" || len(input.Password) > 72 {
			writeError(w, http.StatusBadRequest, "invalid_request", "Identifier and password are required; password must not exceed 72 bytes.")
			return
		}
		result, err := auth.Login(r.Context(), input.Identifier, input.Password)
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "The username/email or password is incorrect.")
		case errors.Is(err, service.ErrInactiveUser):
			writeError(w, http.StatusForbidden, "account_inactive", "This account is inactive. Contact your administrator.")
		case err != nil:
			logger.Error("login failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Unable to sign in right now.")
		default:
			writeJSON(w, http.StatusOK, result)
		}
	})
	router.Group(func(protected chi.Router) {
		protected.Use(requireAuth(auth, logger))
		protected.Get("/auth/me", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"user": r.Context().Value(userContextKey{})})
		})
		protected.Get("/auth/permissions", func(w http.ResponseWriter, r *http.Request) {
			user, _ := requestUser(r)
			writeJSON(w, http.StatusOK, map[string]any{"role": user.Role, "permissions": rolePermissions(user.Role)})
		})
		protected.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			if err := auth.Logout(r.Context(), bearerToken(r)); err != nil {
				logger.Error("logout failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error", "Unable to sign out right now.")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func bearerToken(r *http.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func requireAuth(auth AuthService, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			token := bearerToken(r)
			if token == "" {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
				return
			}
			user, err := auth.Authenticate(r.Context(), token)
			if errors.Is(err, service.ErrUnauthorized) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeError(w, http.StatusUnauthorized, "unauthorized", "Your session is invalid or expired.")
				return
			}
			if err != nil {
				logger.Error("authentication failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error", "Unable to verify your session.")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
		})
	}
}

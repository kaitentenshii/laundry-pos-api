package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

func decodeInput(w http.ResponseWriter, r *http.Request, value any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		writeError(w, 415, "invalid_content_type", "Use application/json.")
		return false
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		writeError(w, 400, "invalid_request", "Provide a valid JSON object.")
		return false
	}
	if d.Decode(&struct{}{}) != io.EOF {
		writeError(w, 400, "invalid_request", "Provide one JSON object.")
		return false
	}
	return true
}
func resourceID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, 400, "invalid_request", "Resource ID must be positive.")
		return 0, false
	}
	return id, true
}
func managementError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var validation *service.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(w, 400, map[string]any{"error": map[string]any{"code": "validation_error", "message": validation.Message, "fields": map[string]string{validation.Field: validation.Message}}})
	case errors.Is(err, service.ErrNotFound):
		writeError(w, 404, "not_found", "Data tidak ditemukan.")
	case errors.Is(err, service.ErrConflict):
		writeError(w, 409, "conflict", "Username atau email sudah digunakan.")
	case errors.Is(err, service.ErrAlreadyPaid):
		writeError(w, 409, "conflict", err.Error())
	case errors.Is(err, service.ErrSelfLockout), errors.Is(err, service.ErrLastAdmin):
		writeError(w, 409, "conflict", err.Error())
	default:
		logger.Error("management operation failed")
		writeError(w, 500, "internal_error", "Unable to process the request.")
	}
}

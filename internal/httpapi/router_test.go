package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := performRequest(http.MethodGet, "/health")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var response healthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "ok" {
		t.Errorf("status body = %q, want ok", response.Status)
	}
}

func TestNotFound(t *testing.T) {
	recorder := performRequest(http.MethodGet, "/missing")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	assertErrorCode(t, recorder, "not_found")
}

func TestMethodNotAllowed(t *testing.T) {
	recorder := performRequest(http.MethodPost, "/health")

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	assertErrorCode(t, recorder, "method_not_allowed")
}

func TestRecoverer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, recorder, "internal_error")
}

func performRequest(method, target string) *httptest.ResponseRecorder {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	request := httptest.NewRequest(method, target, nil)
	recorder := httptest.NewRecorder()
	NewRouter(logger).ServeHTTP(recorder, request)
	return recorder
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()

	var response errorBody
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != want {
		t.Errorf("error code = %q, want %q", response.Error.Code, want)
	}
}

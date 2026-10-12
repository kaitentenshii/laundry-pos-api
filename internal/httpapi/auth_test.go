package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

type authStub struct {
	loginErr   error
	revoked    bool
	loginCalls int
}

func (a *authStub) Login(_ context.Context, identifier, password string) (service.LoginResult, error) {
	a.loginCalls++
	return service.LoginResult{AccessToken: "valid-token", TokenType: "Bearer", ExpiresIn: 3600, User: service.User{ID: 1, Name: "Admin", Role: "admin", PasswordHash: "must-never-be-returned"}}, a.loginErr
}
func (a *authStub) Authenticate(_ context.Context, token string) (service.User, error) {
	if token != "valid-token" || a.revoked {
		return service.User{}, service.ErrUnauthorized
	}
	return service.User{ID: 1, Name: "Admin", Role: "admin", PasswordHash: "must-never-be-returned"}, nil
}
func (a *authStub) Logout(context.Context, string) error { a.revoked = true; return nil }

func authRouter(stub *authStub) http.Handler {
	return NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, RouterOptions{Auth: stub, AllowedOrigins: []string{"http://localhost:5173"}})
}

func authRequest(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestLoginHTTP(t *testing.T) {
	for _, test := range []struct {
		name, body string
		serviceErr error
		status     int
		code       string
	}{
		{"success", `{"identifier":"admin","password":"valid-password"}`, nil, 200, ""},
		{"wrong credentials", `{"identifier":"admin","password":"wrong"}`, service.ErrInvalidCredentials, 401, "invalid_credentials"},
		{"inactive", `{"identifier":"admin","password":"valid-password"}`, service.ErrInactiveUser, 403, "account_inactive"},
		{"missing password", `{"identifier":"admin"}`, nil, 400, "invalid_request"},
		{"malformed", `{`, nil, 400, "invalid_request"},
		{"trailing body", `{"identifier":"admin","password":"valid-password"} {}`, nil, 400, "invalid_request"},
		{"unknown field", `{"identifier":"admin","password":"valid-password","role":"admin"}`, nil, 400, "invalid_request"},
		{"oversized", `{"identifier":"` + strings.Repeat("a", 5000) + `","password":"valid-password"}`, nil, 400, "invalid_request"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stub := &authStub{loginErr: test.serviceErr}
			response := authRequest(authRouter(stub), "POST", "/auth/login", test.body, "")
			if response.Code != test.status {
				t.Fatalf("status=%d want=%d", response.Code, test.status)
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("login response must not be cached")
			}
			if strings.Contains(response.Body.String(), "must-never-be-returned") || strings.Contains(response.Body.String(), "password_hash") {
				t.Fatal("response leaks password hash")
			}
			if test.code != "" {
				assertErrorCode(t, response, test.code)
			}
			if test.status == 400 && stub.loginCalls != 0 {
				t.Fatal("malformed request reached authentication")
			}
			if test.status == 200 {
				var result service.LoginResult
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.User.Role != "admin" || result.AccessToken == "" {
					t.Fatal("login response missing token or role")
				}
			}
		})
	}
}

func TestProtectedRoutesAndLogout(t *testing.T) {
	router := authRouter(&authStub{})
	for _, token := range []string{"", "wrong"} {
		response := authRequest(router, "GET", "/auth/me", "", token)
		if response.Code != 401 || response.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Fatal("protected endpoint allowed unauthenticated access")
		}
	}
	response := authRequest(router, "GET", "/auth/me", "", "valid-token")
	if response.Code != 200 || strings.Contains(response.Body.String(), "password_hash") {
		t.Fatal("invalid protected profile response")
	}
	response = authRequest(router, "POST", "/auth/logout", "", "valid-token")
	if response.Code != 204 {
		t.Fatal("logout failed")
	}
	response = authRequest(router, "GET", "/auth/me", "", "valid-token")
	if response.Code != 401 {
		t.Fatal("revoked token retained access")
	}
}

func TestLoginRequiresJSONAndCORSAllowsOnlyConfiguredOrigin(t *testing.T) {
	router := authRouter(&authStub{})
	request := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 415 {
		t.Fatal("login accepted non-JSON body")
	}
	for _, origin := range []string{"http://localhost:5173", "https://untrusted.example"} {
		request := httptest.NewRequest("OPTIONS", "/auth/login", nil)
		request.Header.Set("Origin", origin)
		request.Header.Set("Access-Control-Request-Method", "POST")
		request.Header.Set("Access-Control-Request-Headers", "content-type, authorization")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if origin == "http://localhost:5173" {
			if response.Code != 204 || response.Header().Get("Access-Control-Allow-Origin") != origin {
				t.Fatal("allowed CORS preflight failed")
			}
		} else if response.Code != 403 || response.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("untrusted origin was permitted")
		}
	}
}

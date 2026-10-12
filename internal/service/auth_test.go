package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type memoryAuthStore struct {
	user   User
	hash   []byte
	expiry time.Time
	err    error
}

func (s *memoryAuthStore) FindUser(_ context.Context, identifier string) (User, error) {
	if s.err != nil {
		return User{}, s.err
	}
	if identifier != s.user.Username && identifier != s.user.Email {
		return User{}, ErrNotFound
	}
	return s.user, nil
}
func (s *memoryAuthStore) CreateSession(_ context.Context, hash []byte, _ int64, expiry time.Time) error {
	s.hash = append([]byte(nil), hash...)
	s.expiry = expiry
	return s.err
}
func (s *memoryAuthStore) SessionUser(_ context.Context, hash []byte, now time.Time) (User, error) {
	if s.err != nil {
		return User{}, s.err
	}
	if string(hash) != string(s.hash) || !s.expiry.After(now) {
		return User{}, ErrNotFound
	}
	return s.user, nil
}
func (s *memoryAuthStore) DeleteSession(context.Context, []byte) error { s.hash = nil; return s.err }

func authFixture(t *testing.T) (*Auth, *memoryAuthStore) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("valid-password-123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryAuthStore{user: User{ID: 1, Name: "Admin", Username: "admin", Email: "admin@example.com", PasswordHash: string(hash), Role: "admin", IsActive: true}}
	auth, err := NewAuth(store, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return auth, store
}

func TestLoginAndSessionLifecycle(t *testing.T) {
	auth, store := authFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	auth.now = func() time.Time { return now }
	for _, identifier := range []string{"admin", " admin@example.com "} {
		result, err := auth.Login(ctx, identifier, "valid-password-123")
		if err != nil {
			t.Fatal(err)
		}
		if result.TokenType != "Bearer" || result.ExpiresIn != 3600 || result.User.Role != "admin" {
			t.Fatalf("unexpected login result: token type=%s ttl=%d role=%s", result.TokenType, result.ExpiresIn, result.User.Role)
		}
		wantHash := sha256.Sum256([]byte(result.AccessToken))
		if string(store.hash) != string(wantHash[:]) || len(store.hash) != 32 {
			t.Fatal("session token must be stored as a SHA-256 hash")
		}
		if user, err := auth.Authenticate(ctx, result.AccessToken); err != nil || user.ID != 1 {
			t.Fatalf("authenticate: %v", err)
		}
		store.user.IsActive = false
		if _, err := auth.Authenticate(ctx, result.AccessToken); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("deactivated users must lose session access")
		}
		store.user.IsActive = true
		now = now.Add(time.Hour)
		if _, err := auth.Authenticate(ctx, result.AccessToken); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("expired sessions must be denied")
		}
	}
	result, err := auth.Login(ctx, "admin", "valid-password-123")
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Logout(ctx, result.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, result.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("logged-out tokens must be denied")
	}
}

func TestLoginRejectsInvalidAndInactiveAccounts(t *testing.T) {
	auth, store := authFixture(t)
	for _, test := range []struct{ identifier, password string }{{"admin", "wrong-password"}, {"missing", "valid-password-123"}, {"", "valid-password-123"}, {"admin", ""}} {
		if _, err := auth.Login(context.Background(), test.identifier, test.password); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
		if store.hash != nil {
			t.Fatal("rejected login created a session")
		}
	}
	store.user.IsActive = false
	if _, err := auth.Login(context.Background(), "admin", "valid-password-123"); !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("expected inactive account error, got %v", err)
	}
	store.user.IsActive = true
	store.user.PasswordHash = "plaintext-password"
	if _, err := auth.Login(context.Background(), "admin", "plaintext-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("plaintext password must never authenticate")
	}
}

func TestAuthPropagatesStorageFailureAndRejectsMalformedTokens(t *testing.T) {
	auth, store := authFixture(t)
	store.err = errors.New("database unavailable")
	if _, err := auth.Login(context.Background(), "admin", "valid-password-123"); !errors.Is(err, store.err) {
		t.Fatal("database error was hidden")
	}
	for _, token := range []string{"", "invalid", "../../../", "YWJj"} {
		if _, err := auth.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthorized) {
			t.Fatal("malformed token was accepted")
		}
	}
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func TestPostgresAuthentication(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to an isolated migrated PostgreSQL database")
	}
	ctx := context.Background()
	pool, err := Open(ctx, databaseURL, 5*time.Second)
	if err != nil {
		t.Fatal("connect to test database")
	}
	defer pool.Close()
	username := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	email := username + "@example.com"
	password := "valid-password-123"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	var userID int64
	err = pool.QueryRow(ctx, `INSERT INTO laundry.users (name, username, email, password_hash, role) VALUES ($1,$2,$3,$4,$5) RETURNING id`, "Test Cashier", username, email, string(passwordHash), "cashier").Scan(&userID)
	if err != nil {
		t.Fatal("insert test user: ", err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, `DELETE FROM laundry.users WHERE id=$1`, userID); err != nil {
			t.Error("cleanup test user")
		}
	}()
	auth, err := service.NewAuth(NewAuthStore(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, identifier := range []string{strings.ToUpper(username), strings.ToUpper(email)} {
		result, err := auth.Login(ctx, identifier, password)
		if err != nil || result.User.ID != userID || result.User.Role != "cashier" {
			t.Fatalf("username/email login failed: %v", err)
		}
		var storedTokenBytes int
		if err := pool.QueryRow(ctx, `SELECT octet_length(token_hash) FROM laundry.auth_sessions WHERE user_id=$1 LIMIT 1`, userID).Scan(&storedTokenBytes); err != nil || storedTokenBytes != 32 {
			t.Fatal("database did not store hashed token")
		}
		if _, err := auth.Authenticate(ctx, result.AccessToken); err != nil {
			t.Fatal(err)
		}
		if err := auth.Logout(ctx, result.AccessToken); err != nil {
			t.Fatal(err)
		}
		if _, err := auth.Authenticate(ctx, result.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
			t.Fatal("revoked token retained database access")
		}
	}
	if _, err := auth.Login(ctx, username, "incorrect"); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatal("wrong password accepted")
	}
	first, err := auth.Login(ctx, username, password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := auth.Login(ctx, username, password)
	if err != nil {
		t.Fatal(err)
	}
	if first.AccessToken == second.AccessToken {
		t.Fatal("separate logins reused a token")
	}
	if err := auth.Logout(ctx, first.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, first.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatal("logout failed to revoke its session")
	}
	if _, err := auth.Authenticate(ctx, second.AccessToken); err != nil {
		t.Fatal("logout incorrectly revoked another device session")
	}
	result, err := auth.Login(ctx, username, password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE laundry.auth_sessions SET expires_at = CURRENT_TIMESTAMP - INTERVAL '1 second' WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, result.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatal("expired token accepted")
	}
	result, err = auth.Login(ctx, username, password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE laundry.users SET is_active=false WHERE id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Login(ctx, username, password); !errors.Is(err, service.ErrInactiveUser) {
		t.Fatal("inactive user logged in")
	}
	if _, err := auth.Authenticate(ctx, result.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatal("inactive user retained access")
	}
}

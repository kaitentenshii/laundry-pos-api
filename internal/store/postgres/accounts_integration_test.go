package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kaitentenshii/laundry-pos-api/internal/service"
)

func TestAccountManagementPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL integration")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url, 5*time.Second)
	if err != nil {
		t.Fatal("connect to test database")
	}
	defer pool.Close()
	manager := service.NewAccounts(NewAccountStore(pool))
	active, inactive := true, false
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	admin, err := manager.Create(ctx, service.CreateAccountInput{AccountInput: service.AccountInput{Name: "Test Admin", Identifier: "a_" + suffix, Role: "admin", IsActive: &active}, Password: "password1234"})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM laundry.users WHERE id=$1`, admin.ID)
	cashierInput := service.CreateAccountInput{AccountInput: service.AccountInput{Name: "Test Cashier", Identifier: "c_" + suffix + "@test.example", Role: "cashier", IsActive: &active}, Password: "password1234"}
	cashier, err := manager.Create(ctx, cashierInput)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM laundry.users WHERE id=$1`, cashier.ID)
	if _, err := manager.Create(ctx, cashierInput); !errors.Is(err, service.ErrConflict) {
		t.Fatal("duplicate email accepted")
	}
	auth, err := service.NewAuth(NewAuthStore(pool), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Login(ctx, admin.Username, "password1234"); err != nil {
		t.Fatal("username-only account could not login")
	}
	login, err := auth.Login(ctx, cashier.Email, "password1234")
	if err != nil {
		t.Fatal(err)
	}
	data, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range data {
		if a.ID == cashier.ID && a.LastActiveAt != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("last activity was not recorded")
	}
	profile := service.AccountInput{Name: "Updated Cashier", Identifier: cashier.Email, Role: "cashier", IsActive: &inactive}
	updated, err := manager.Update(ctx, admin.ID, cashier.ID, profile)
	if err != nil || updated.IsActive || updated.Name != "Updated Cashier" {
		t.Fatal("account update failed")
	}
	if _, err := auth.Authenticate(ctx, login.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatal("inactive account retained its session")
	}
	if _, err := auth.Login(ctx, cashier.Email, "password1234"); !errors.Is(err, service.ErrInactiveUser) {
		t.Fatal("inactive account logged in")
	}
	profile.IsActive = &active
	if _, err := manager.Update(ctx, admin.ID, cashier.ID, profile); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, login.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatal("reactivation restored a revoked session")
	}
	login, err = auth.Login(ctx, cashier.Email, "password1234")
	if err != nil {
		t.Fatal(err)
	}
	profile.Role = "admin"
	if _, err := manager.Update(ctx, admin.ID, cashier.ID, profile); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Authenticate(ctx, login.AccessToken); !errors.Is(err, service.ErrUnauthorized) {
		t.Fatal("role change failed to revoke sessions")
	}
	if _, err := manager.Update(ctx, admin.ID, admin.ID, service.AccountInput{Name: admin.Name, Identifier: admin.Username, Role: "cashier", IsActive: &active}); !errors.Is(err, service.ErrSelfLockout) {
		t.Fatal("self-demotion was accepted")
	}
}

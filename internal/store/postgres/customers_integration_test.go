package postgres

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"os"
	"testing"
	"time"
)

func TestCustomerRegistrationPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url, 5*time.Second)
	if err != nil {
		t.Fatal("connect to test database")
	}
	defer pool.Close()
	manager := service.NewCustomers(NewCustomerStore(pool))
	input := service.CustomerInput{Name: "Customer integration", Whatsapp: "081234567890", Address: "Jl. Melati", Notes: "Tanpa pewangi"}
	first, err := manager.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM laundry.customers WHERE id=$1`, first.ID)
	if first.ID == 0 || first.Whatsapp != "+6281234567890" || first.WhatsappOptIn || first.Notes != input.Notes || first.CreatedAt.IsZero() {
		t.Fatal("incorrect persisted customer")
	}
	input.WhatsappOptIn = true
	second, err := manager.Create(ctx, input)
	if err != nil {
		t.Fatal("shared household phone must be allowed", err)
	}
	defer pool.Exec(ctx, `DELETE FROM laundry.customers WHERE id=$1`, second.ID)
	if second.ID == first.ID || !second.WhatsappOptIn {
		t.Fatal("independent customer consent not persisted")
	}
	data, err := manager.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range data {
		if c.ID == first.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("customer missing from list")
	}
}

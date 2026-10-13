package postgres

import (
	"context"
	"errors"
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
	data, err := manager.List(ctx, service.CustomerQuery{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range data.Data {
		if c.ID == first.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("customer missing from list")
	}
	input.Name = "Updated customer"
	input.Whatsapp = "081111111111"
	input.Address = "New address"
	input.Notes = "New notes"
	input.WhatsappOptIn = true
	updated, err := manager.Update(ctx, first.ID, input)
	if err != nil || updated.ID != first.ID || !updated.CreatedAt.Equal(first.CreatedAt) || updated.Name != input.Name || updated.Whatsapp != "+6281111111111" || !updated.WhatsappOptIn {
		t.Fatal("update changed identity or did not persist fields", err)
	}
	if _, err := manager.Update(ctx, 9223372036854775807, input); !errors.Is(err, service.ErrNotFound) {
		t.Fatal("missing customer not detected")
	}
	input.Name = "Literal_%_Customer"
	unique, err := manager.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM laundry.customers WHERE id=$1`, unique.ID)
	results, err := manager.List(ctx, service.CustomerQuery{Search: "literal_%_", PageSize: 1})
	if err != nil || results.Total != 1 || len(results.Data) != 1 || results.Data[0].ID != unique.ID {
		t.Fatal("literal/case-insensitive search failed", err)
	}
	results, err = manager.List(ctx, service.CustomerQuery{Search: "literal_%_", Page: 2, PageSize: 1})
	if err != nil || results.Total != 1 || len(results.Data) != 0 {
		t.Fatal("pagination totals failed", err)
	}
	consent := false
	results, err = manager.List(ctx, service.CustomerQuery{Search: "literal_%_", Consent: &consent})
	if err != nil || results.Total != 0 {
		t.Fatal("consent filter ignored", err)
	}
	results, err = manager.List(ctx, service.CustomerQuery{Search: "081111111111"})
	if err != nil || results.Total < 2 {
		t.Fatal("local phone search failed", err)
	}
}

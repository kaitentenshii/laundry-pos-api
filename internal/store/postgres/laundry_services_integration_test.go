package postgres

import (
	"context"
	"errors"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"os"
	"testing"
	"time"
)

func TestLaundryServicesPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := Open(ctx, url, 5*time.Second)
	if err != nil {
		t.Fatal("connect test database")
	}
	defer pool.Close()
	manager := service.NewLaundryServices(NewLaundryServiceStore(pool))
	price := int64(8000)
	active := true
	input := service.LaundryServiceInput{Name: "Test service", Unit: "KG", Price: &price, EstimatedHours: 48, IsActive: &active}
	created, err := manager.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM laundry.services WHERE id=$1`, created.ID)
	input.Name = "Updated service"
	input.Unit = "ITEM"
	input.EstimatedHours = 6
	active = false
	updated, err := manager.Update(ctx, created.ID, input)
	if err != nil || updated.IsActive || updated.Unit != "ITEM" || !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatal("update failed", err)
	}
	data, err := manager.List(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range data {
		if record.ID == created.ID {
			t.Fatal("cashier active selection exposed inactive service")
		}
	}
	data, err = manager.List(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range data {
		if record.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("inactive service deleted")
	}
	if _, err := manager.Update(ctx, 9223372036854775807, input); !errors.Is(err, service.ErrNotFound) {
		t.Fatal("missing service not detected")
	}
}

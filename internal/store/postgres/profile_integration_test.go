package postgres

import (
	"context"
	"github.com/kaitentenshii/laundry-pos-api/internal/service"
	"testing"
)

func TestProfilePersistsAndSingleton(t *testing.T) {
	f := setupOrders(t)
	ctx := context.Background()
	store := NewProfileStore(f.pool)
	before, err := store.GetProfile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := store.SaveProfile(ctx, before.ProfileInput); err != nil {
			t.Error(err)
		}
	})
	manager := service.NewProfile(store)
	want := service.ProfileInput{Name: " Laundry Test ", Contact: "081234567890", OperatingHours: "08.00–20.00", Address: " Jalan Melati "}
	saved, err := manager.Save(ctx, want)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetProfile(ctx)
	if err != nil || loaded.Name != "Laundry Test" || loaded.Address != "Jalan Melati" || loaded.Contact != "+6281234567890" || loaded.UpdatedAt.Before(saved.UpdatedAt) {
		t.Fatal("save/read failure", loaded, err)
	}
	if _, err = f.pool.Exec(ctx, `INSERT INTO laundry.profile(id) VALUES(false)`); err == nil {
		t.Fatal("second profile accepted")
	}
}

package service

import "testing"

func TestLaundryServiceProfile(t *testing.T) {
	price := int64(8000)
	active := true
	input := LaundryServiceInput{Name: " Cuci + setrika ", Unit: "KG", Price: &price, EstimatedHours: 48, IsActive: &active}
	profile, err := laundryServiceProfile(input, false)
	if err != nil || profile.Name != "Cuci + setrika" || profile.Price != 8000 {
		t.Fatal("valid profile rejected")
	}
	input.Unit = "pcs"
	if _, err := laundryServiceProfile(input, false); err == nil {
		t.Fatal("UI unit label accepted as API enum")
	}
	input.Unit = "ITEM"
	price = -1
	if _, err := laundryServiceProfile(input, false); err == nil {
		t.Fatal("negative price accepted")
	}
	price = 0
	input.EstimatedHours = 0
	if _, err := laundryServiceProfile(input, false); err == nil {
		t.Fatal("zero estimate accepted")
	}
	input.EstimatedHours = 6
	input.IsActive = nil
	if _, err := laundryServiceProfile(input, false); err == nil {
		t.Fatal("missing update status accepted")
	}
	profile, err = laundryServiceProfile(input, true)
	if err != nil || !profile.IsActive || profile.Unit != "ITEM" {
		t.Fatal("create defaults invalid")
	}
}

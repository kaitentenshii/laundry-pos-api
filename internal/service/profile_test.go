package service

import (
	"strings"
	"testing"
)

func TestNormalizeLaundryProfile(t *testing.T) {
	good := ProfileInput{Name: " Laundry Mitra ", Contact: "0812-0000-1000", OperatingHours: " 08.00–20.00 ", Address: " Jalan Melati "}
	p, err := NormalizeProfile(good)
	if err != nil || p.Name != "Laundry Mitra" || p.Contact != "+6281200001000" || p.Address != "Jalan Melati" {
		t.Fatal("profile not normalized", p, err)
	}
	for _, field := range []string{"name", "contact", "operating_hours", "address"} {
		bad := good
		switch field {
		case "name":
			bad.Name = " "
		case "contact":
			bad.Contact = "invalid"
		case "operating_hours":
			bad.OperatingHours = strings.Repeat("a", 121)
		case "address":
			bad.Address = strings.Repeat("a", 1001)
		}
		if _, err := NormalizeProfile(bad); err == nil {
			t.Fatal("invalid field accepted", field)
		}
	}
	good.Contact = "021 12345678"
	if p, err := NormalizeProfile(good); err != nil || p.Contact != "+622112345678" {
		t.Fatal("business landline rejected", p, err)
	}
}

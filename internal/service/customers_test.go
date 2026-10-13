package service

import "testing"

func TestCustomerValidation(t *testing.T) {
	for _, input := range []string{"0812-3456-7890", "62 812 3456 7890", "+62 (812) 3456-7890"} {
		phone, err := NormalizeWhatsapp(input)
		if err != nil || phone != "+6281234567890" {
			t.Fatalf("normalization %q: %q %v", input, phone, err)
		}
	}
	for _, input := range []string{"", "123", "+0123456789", "0812letters3456", "++6281234567890", "+1234567890123456"} {
		if _, err := NormalizeWhatsapp(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	if _, err := customerProfile(CustomerInput{Name: " ", Whatsapp: "081234567890"}); err == nil {
		t.Fatal("empty name accepted")
	}
	profile, err := customerProfile(CustomerInput{Name: " Sari ", Whatsapp: "081234567890"})
	if err != nil || profile.Name != "Sari" || profile.WhatsappOptIn {
		t.Fatal("default consent/profile invalid")
	}
}

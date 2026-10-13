package service

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type CustomerInput struct {
	Name          string `json:"name"`
	Whatsapp      string `json:"whatsapp"`
	Address       string `json:"address"`
	Notes         string `json:"notes"`
	WhatsappOptIn bool   `json:"whatsapp_opt_in"`
}
type Customer struct {
	CustomerInput
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type CustomerStore interface {
	ListCustomers(context.Context) ([]Customer, error)
	InsertCustomer(context.Context, CustomerInput) (Customer, error)
	UpdateCustomer(context.Context, int64, CustomerInput) (Customer, error)
}
type Customers struct{ store CustomerStore }

func NewCustomers(store CustomerStore) *Customers { return &Customers{store} }

var internationalPhone = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func NormalizeWhatsapp(value string) (string, error) {
	value = strings.TrimSpace(value)
	for _, c := range value {
		if !strings.ContainsRune("0123456789+ -()", c) {
			return "", Invalid("whatsapp", "Masukkan nomor WhatsApp yang valid.")
		}
	}
	value = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(value)
	if strings.HasPrefix(value, "08") {
		value = "+62" + value[1:]
	} else if strings.HasPrefix(value, "62") {
		value = "+" + value
	}
	if !internationalPhone.MatchString(value) {
		return "", Invalid("whatsapp", "Gunakan nomor 08… atau format internasional +62… (8–15 digit).")
	}
	return value, nil
}
func customerProfile(input CustomerInput) (CustomerInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Address = strings.TrimSpace(input.Address)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 120 {
		return input, Invalid("name", "Nama wajib diisi, maksimal 120 karakter.")
	}
	if utf8.RuneCountInString(input.Address) > 1000 {
		return input, Invalid("address", "Alamat maksimal 1000 karakter.")
	}
	if utf8.RuneCountInString(input.Notes) > 2000 {
		return input, Invalid("notes", "Catatan maksimal 2000 karakter.")
	}
	phone, err := NormalizeWhatsapp(input.Whatsapp)
	input.Whatsapp = phone
	return input, err
}
func (s *Customers) List(ctx context.Context) ([]Customer, error) { return s.store.ListCustomers(ctx) }
func (s *Customers) Create(ctx context.Context, input CustomerInput) (Customer, error) {
	input, err := customerProfile(input)
	if err != nil {
		return Customer{}, err
	}
	return s.store.InsertCustomer(ctx, input)
}

func (s *Customers) Update(ctx context.Context, id int64, input CustomerInput) (Customer, error) {
	input, err := customerProfile(input)
	if err != nil {
		return Customer{}, err
	}
	return s.store.UpdateCustomer(ctx, id, input)
}

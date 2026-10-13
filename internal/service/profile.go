package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

type ProfileInput struct {
	Name           string `json:"name"`
	Contact        string `json:"contact"`
	OperatingHours string `json:"operating_hours"`
	Address        string `json:"address"`
}
type LaundryProfile struct {
	ProfileInput
	UpdatedAt time.Time `json:"updated_at"`
}
type ProfileStore interface {
	GetProfile(context.Context) (LaundryProfile, error)
	SaveProfile(context.Context, ProfileInput) (LaundryProfile, error)
}
type ProfileService struct{ store ProfileStore }

func NewProfile(store ProfileStore) *ProfileService { return &ProfileService{store} }
func (s *ProfileService) Get(ctx context.Context) (LaundryProfile, error) {
	return s.store.GetProfile(ctx)
}
func NormalizeProfile(p ProfileInput) (ProfileInput, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.Contact = strings.TrimSpace(p.Contact)
	p.OperatingHours = strings.TrimSpace(p.OperatingHours)
	p.Address = strings.TrimSpace(p.Address)
	for _, f := range []struct {
		name, value string
		max         int
	}{{"name", p.Name, 120}, {"operating_hours", p.OperatingHours, 120}, {"address", p.Address, 1000}} {
		if f.value == "" || utf8.RuneCountInString(f.value) > f.max {
			return p, Invalid(f.name, "Wajib diisi dan tidak boleh melebihi batas panjang.")
		}
	}
	if strings.HasPrefix(p.Contact, "0") {
		p.Contact = "+62" + p.Contact[1:]
	}
	phone, err := NormalizeWhatsapp(p.Contact)
	if err != nil {
		return p, Invalid("contact", "Gunakan nomor kontak yang valid: 08…, 02…, atau +62….")
	}
	p.Contact = phone
	return p, nil
}
func (s *ProfileService) Save(ctx context.Context, p ProfileInput) (LaundryProfile, error) {
	p, err := NormalizeProfile(p)
	if err != nil {
		return LaundryProfile{}, err
	}
	return s.store.SaveProfile(ctx, p)
}

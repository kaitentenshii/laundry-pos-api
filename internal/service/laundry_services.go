package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
)

type LaundryServiceInput struct {
	Name           string `json:"name"`
	Unit           string `json:"unit"`
	Price          *int64 `json:"price"`
	EstimatedHours int    `json:"estimated_hours"`
	IsActive       *bool  `json:"is_active"`
}
type LaundryService struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Unit           string    `json:"unit"`
	Price          int64     `json:"price"`
	EstimatedHours int       `json:"estimated_hours"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type LaundryServiceStore interface {
	ListServices(context.Context, bool) ([]LaundryService, error)
	SaveService(context.Context, int64, LaundryService) (LaundryService, error)
}
type LaundryServices struct{ store LaundryServiceStore }

func NewLaundryServices(store LaundryServiceStore) *LaundryServices { return &LaundryServices{store} }
func (s *LaundryServices) List(ctx context.Context, activeOnly bool) ([]LaundryService, error) {
	return s.store.ListServices(ctx, activeOnly)
}
func laundryServiceProfile(input LaundryServiceInput, creating bool) (LaundryService, error) {
	profile := LaundryService{Name: strings.TrimSpace(input.Name), Unit: input.Unit, EstimatedHours: input.EstimatedHours, IsActive: true}
	if profile.Name == "" || utf8.RuneCountInString(profile.Name) > 120 {
		return profile, Invalid("name", "Nama layanan wajib diisi, maksimal 120 karakter.")
	}
	if input.Unit != "KG" && input.Unit != "ITEM" {
		return profile, Invalid("unit", "Pilih satuan kg atau pcs.")
	}
	if input.Price == nil || *input.Price < 0 || *input.Price > 999999999 {
		return profile, Invalid("price", "Harga wajib berupa rupiah bulat antara 0 dan 999.999.999.")
	}
	profile.Price = *input.Price
	if input.EstimatedHours < 1 || input.EstimatedHours > 8760 {
		return profile, Invalid("estimated_hours", "Estimasi wajib antara 1 dan 8760 jam.")
	}
	if input.IsActive != nil {
		profile.IsActive = *input.IsActive
	} else if !creating {
		return profile, Invalid("is_active", "Status layanan wajib dipilih.")
	}
	return profile, nil
}
func (s *LaundryServices) Create(ctx context.Context, input LaundryServiceInput) (LaundryService, error) {
	profile, err := laundryServiceProfile(input, true)
	if err != nil {
		return profile, err
	}
	return s.store.SaveService(ctx, 0, profile)
}
func (s *LaundryServices) Update(ctx context.Context, id int64, input LaundryServiceInput) (LaundryService, error) {
	profile, err := laundryServiceProfile(input, false)
	if err != nil {
		return profile, err
	}
	return s.store.SaveService(ctx, id, profile)
}

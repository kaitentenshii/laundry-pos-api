package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var ErrConflict = errors.New("data conflicts with an existing record")
var ErrSelfLockout = errors.New("akun yang sedang digunakan tidak boleh dinonaktifkan atau diturunkan perannya")
var ErrLastAdmin = errors.New("administrator aktif terakhir tidak dapat dinonaktifkan atau diturunkan perannya")

type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string  { return e.Message }
func Invalid(field, message string) error { return &ValidationError{Field: field, Message: message} }

type Account struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	IsActive     bool       `json:"is_active"`
	LastActiveAt *time.Time `json:"last_active_at"`
}
type AccountInput struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	Role       string `json:"role"`
	IsActive   *bool  `json:"is_active"`
}
type CreateAccountInput struct {
	AccountInput
	Password string `json:"password"`
}
type AccountStore interface {
	ListAccounts(context.Context) ([]Account, error)
	InsertAccount(context.Context, Account, string) (Account, error)
	UpdateAccount(context.Context, int64, int64, Account) (Account, error)
}
type Accounts struct{ store AccountStore }

func NewAccounts(store AccountStore) *Accounts                  { return &Accounts{store: store} }
func (s *Accounts) List(ctx context.Context) ([]Account, error) { return s.store.ListAccounts(ctx) }

var accountUsername = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)

func accountProfile(input AccountInput, creating bool) (Account, error) {
	name, identifier := strings.TrimSpace(input.Name), strings.ToLower(strings.TrimSpace(input.Identifier))
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return Account{}, Invalid("name", "Nama wajib diisi, maksimal 120 karakter.")
	}
	if input.Role != "admin" && input.Role != "cashier" {
		return Account{}, Invalid("role", "Peran harus Admin atau Kasir.")
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	} else if !creating {
		return Account{}, Invalid("is_active", "Status akun wajib dipilih.")
	}
	a := Account{Name: name, Role: input.Role, IsActive: active}
	if strings.Contains(identifier, "@") {
		address, err := mail.ParseAddress(identifier)
		if err != nil || address.Address != identifier || len(identifier) > 320 {
			return Account{}, Invalid("identifier", "Masukkan email yang valid.")
		}
		a.Email = identifier
		if creating {
			raw := make([]byte, 12)
			if _, err := rand.Read(raw); err != nil {
				return Account{}, err
			}
			a.Username = "u_" + hex.EncodeToString(raw)
		}
	} else {
		if !accountUsername.MatchString(identifier) {
			return Account{}, Invalid("identifier", "Username menggunakan 3–64 huruf, angka, titik, garis bawah, atau tanda hubung.")
		}
		a.Username = identifier
	}
	return a, nil
}
func (s *Accounts) Create(ctx context.Context, input CreateAccountInput) (Account, error) {
	a, err := accountProfile(input.AccountInput, true)
	if err != nil {
		return Account{}, err
	}
	if utf8.RuneCountInString(input.Password) < 8 || len(input.Password) > 72 {
		return Account{}, Invalid("password", "Kata sandi minimal 8 karakter dan maksimal 72 byte.")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, err
	}
	return s.store.InsertAccount(ctx, a, string(hash))
}
func (s *Accounts) Update(ctx context.Context, actorID, id int64, input AccountInput) (Account, error) {
	a, err := accountProfile(input, false)
	if err != nil {
		return Account{}, err
	}
	return s.store.UpdateAccount(ctx, actorID, id, a)
}

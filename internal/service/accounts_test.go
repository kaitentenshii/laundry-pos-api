package service

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

type accountStoreStub struct {
	saved Account
	hash  string
}

func (s *accountStoreStub) ListAccounts(context.Context) ([]Account, error) {
	return []Account{s.saved}, nil
}
func (s *accountStoreStub) InsertAccount(_ context.Context, a Account, h string) (Account, error) {
	s.saved = a
	s.hash = h
	return a, nil
}
func (s *accountStoreStub) UpdateAccount(_ context.Context, _, _ int64, a Account) (Account, error) {
	s.saved = a
	return a, nil
}
func TestCreateAccountValidationAndHashing(t *testing.T) {
	store := &accountStoreStub{}
	manager := NewAccounts(store)
	active := true
	input := CreateAccountInput{AccountInput: AccountInput{Name: "Kasir", Identifier: "CASHIER@Test.Example", Role: "cashier", IsActive: &active}, Password: "password1234"}
	account, err := manager.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if account.Email != "cashier@test.example" || account.Username == "" {
		t.Fatal("email identity was not normalized")
	}
	if store.hash == input.Password || bcrypt.CompareHashAndPassword([]byte(store.hash), []byte(input.Password)) != nil {
		t.Fatal("password was not securely hashed")
	}
	for _, change := range []func(*CreateAccountInput){func(i *CreateAccountInput) { i.Name = " " }, func(i *CreateAccountInput) { i.Identifier = "invalid email@" }, func(i *CreateAccountInput) { i.Role = "owner" }, func(i *CreateAccountInput) { i.Password = "short" }} {
		bad := input
		change(&bad)
		if _, err := manager.Create(context.Background(), bad); err == nil {
			t.Fatal("invalid account accepted")
		}
	}
}

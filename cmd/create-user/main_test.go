package main

import (
	"strings"
	"testing"
)

func TestUserValidation(t *testing.T) {
	valid := input{name: "Admin", username: "admin", email: "admin@example.com", role: "admin", password: "secure-password-123"}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*input){
		func(i *input) { i.username = "admin@example.com" },
		func(i *input) { i.email = "invalid" },
		func(i *input) { i.role = "unknown" },
		func(i *input) { i.password = "short" },
		func(i *input) { i.password = strings.Repeat("x", 73) },
		func(i *input) { i.name = " " },
	} {
		data := valid
		change(&data)
		if err := data.validate(); err == nil {
			t.Fatal("invalid account accepted")
		}
	}
}

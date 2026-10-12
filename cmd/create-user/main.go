package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kaitentenshii/laundry-pos-api/internal/config"
	"github.com/kaitentenshii/laundry-pos-api/internal/store/postgres"
	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{3,64}$`)

type input struct{ name, username, email, role, password string }

func (i input) validate() error {
	if strings.TrimSpace(i.name) == "" || !usernamePattern.MatchString(i.username) {
		return errors.New("provide a name and a 3–64 character username using letters, numbers, dot, underscore, or hyphen")
	}
	address, err := mail.ParseAddress(i.email)
	if err != nil || address.Address != i.email || len(i.email) > 320 {
		return errors.New("provide a valid email address")
	}
	if i.role != "admin" && i.role != "cashier" {
		return errors.New("role must be admin or cashier")
	}
	if len(i.password) < 12 || len(i.password) > 72 {
		return errors.New("LAUNDRY_USER_PASSWORD must contain 12–72 bytes")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var data input
	flag.StringVar(&data.name, "name", "", "Display name")
	flag.StringVar(&data.username, "username", "", "Username")
	flag.StringVar(&data.email, "email", "", "Email address")
	flag.StringVar(&data.role, "role", "cashier", "admin or cashier")
	flag.Parse()
	data.password = os.Getenv("LAUNDRY_USER_PASSWORD")
	data.name = strings.TrimSpace(data.name)
	data.username = strings.ToLower(strings.TrimSpace(data.username))
	data.email = strings.ToLower(strings.TrimSpace(data.email))
	if err := data.validate(); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := postgres.Open(ctx, cfg.DatabaseURL, cfg.DatabaseConnectTimeout)
	if err != nil {
		return err
	}
	defer pool.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte(data.password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("unable to hash password")
	}
	var id int64
	err = pool.QueryRow(ctx, `INSERT INTO laundry.users (name, username, email, password_hash, role) VALUES ($1, $2, $3, $4, $5) RETURNING id`, data.name, data.username, data.email, string(hash), data.role).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errors.New("username or email already exists")
		}
		return errors.New("unable to create user; ensure database migrations have been applied")
	}
	fmt.Printf("Created user %d (%s).\n", id, data.role)
	return nil
}

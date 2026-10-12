package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound           = errors.New("user or session not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveUser       = errors.New("user is inactive")
	ErrUnauthorized       = errors.New("authentication required")
)

type User struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
	IsActive     bool   `json:"-"`
}

type AuthStore interface {
	FindUser(context.Context, string) (User, error)
	CreateSession(context.Context, []byte, int64, time.Time) error
	SessionUser(context.Context, []byte, time.Time) (User, error)
	DeleteSession(context.Context, []byte) error
}

type LoginResult struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	User        User   `json:"user"`
}

type Auth struct {
	store     AuthStore
	ttl       time.Duration
	dummyHash []byte
	now       func() time.Time
}

func NewAuth(store AuthStore, ttl time.Duration) (*Auth, error) {
	if ttl < time.Second || ttl > 24*time.Hour {
		return nil, errors.New("session duration must be between 1 second and 24 hours")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("not-a-user-password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Auth{store: store, ttl: ttl, dummyHash: hash, now: time.Now}, nil
}

func (a *Auth) Login(ctx context.Context, identifier, password string) (LoginResult, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || len(identifier) > 320 || password == "" || len(password) > 72 {
		return LoginResult{}, ErrInvalidCredentials
	}
	user, err := a.store.FindUser(ctx, identifier)
	if errors.Is(err, ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(a.dummyHash, []byte(password))
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !user.IsActive {
		return LoginResult{}, ErrInactiveUser
	}
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return LoginResult{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token))
	ttl := time.Duration(int64(a.ttl/time.Second)) * time.Second
	if err := a.store.CreateSession(ctx, hash[:], user.ID, a.now().Add(ttl)); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{AccessToken: token, TokenType: "Bearer", ExpiresIn: int64(ttl / time.Second), User: user}, nil
}

func (a *Auth) Authenticate(ctx context.Context, token string) (User, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return User{}, ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(token))
	user, err := a.store.SessionUser(ctx, hash[:], a.now())
	if errors.Is(err, ErrNotFound) {
		return User{}, ErrUnauthorized
	}
	if err != nil {
		return User{}, err
	}
	if !user.IsActive {
		return User{}, ErrUnauthorized
	}
	return user, nil
}

func (a *Auth) Logout(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	return a.store.DeleteSession(ctx, hash[:])
}

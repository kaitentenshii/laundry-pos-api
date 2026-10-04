package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/test")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want development", cfg.Environment)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.DatabaseConnectTimeout != 5*time.Second {
		t.Errorf("DatabaseConnectTimeout = %s, want 5s", cfg.DatabaseConnectTimeout)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "9090")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/test")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.Environment != "test" || cfg.Port != 9090 || cfg.ShutdownTimeout != 3*time.Second || cfg.DatabaseConnectTimeout != 2*time.Second || cfg.DatabaseURL != "postgres://user:password@localhost:5432/test" {
		t.Fatalf("Load() = %+v, want environment=test port=9090 shutdown timeout=3s database timeout=2s", cfg)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/test")
	t.Setenv("PORT", "70000")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned no error for an invalid port")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned no error when DATABASE_URL was missing")
	}
}

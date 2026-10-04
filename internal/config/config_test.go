package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")

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
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "9090")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.Environment != "test" || cfg.Port != 9090 || cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("Load() = %+v, want environment=test port=9090 timeout=3s", cfg)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("PORT", "70000")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned no error for an invalid port")
	}
}

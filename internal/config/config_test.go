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
	t.Setenv("MIGRATIONS_PATH", "")
	t.Setenv("AUTH_SESSION_TTL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

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
	if cfg.MigrationsPath != "file://migrations" {
		t.Errorf("MigrationsPath = %q, want file://migrations", cfg.MigrationsPath)
	}
	if cfg.AuthSessionTTL != 8*time.Hour || len(cfg.AllowedOrigins) != 2 {
		t.Fatal("unexpected authentication defaults")
	}
}

func TestLoadAuthConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/test")
	t.Setenv("AUTH_SESSION_TTL", "30m")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://laundry.example,http://localhost:5173")
	cfg, err := Load()
	if err != nil || cfg.AuthSessionTTL != 30*time.Minute || cfg.AllowedOrigins[0] != "https://laundry.example" {
		t.Fatalf("invalid auth configuration: %v", err)
	}
	for _, value := range []string{"0s", "500ms", "25h", "invalid"} {
		t.Setenv("AUTH_SESSION_TTL", value)
		if _, err := Load(); err == nil {
			t.Fatalf("accepted invalid session duration %s", value)
		}
	}
	t.Setenv("AUTH_SESSION_TTL", "8h")
	for _, value := range []string{"*", "http://*", "https://laundry.example/path", "file://example", "https://user:password@example.com"} {
		t.Setenv("CORS_ALLOWED_ORIGINS", value)
		if _, err := Load(); err == nil {
			t.Fatal("accepted invalid CORS origin")
		}
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "9090")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/test")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "2s")
	t.Setenv("MIGRATIONS_PATH", "file://testdata/migrations")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.Environment != "test" || cfg.Port != 9090 || cfg.ShutdownTimeout != 3*time.Second || cfg.DatabaseConnectTimeout != 2*time.Second || cfg.DatabaseURL != "postgres://user:password@localhost:5432/test" || cfg.MigrationsPath != "file://testdata/migrations" {
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

package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEnvironment     = "development"
	defaultPort            = 8080
	defaultShutdownTimeout = 10 * time.Second
	defaultMigrationsPath  = "file://migrations"
)

type Config struct {
	Environment            string
	Port                   int
	ShutdownTimeout        time.Duration
	DatabaseURL            string
	DatabaseConnectTimeout time.Duration
	MigrationsPath         string
	AuthSessionTTL         time.Duration
	AllowedOrigins         []string
}

func Load() (Config, error) {
	port, err := readPort(os.Getenv("PORT"))
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := readDuration("SHUTDOWN_TIMEOUT", os.Getenv("SHUTDOWN_TIMEOUT"), defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	databaseConnectTimeout, err := readDuration("DATABASE_CONNECT_TIMEOUT", os.Getenv("DATABASE_CONNECT_TIMEOUT"), 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = defaultMigrationsPath
	}

	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = defaultEnvironment
	}
	authSessionTTL, err := readDuration("AUTH_SESSION_TTL", os.Getenv("AUTH_SESSION_TTL"), 8*time.Hour)
	if err != nil || authSessionTTL < time.Second || authSessionTTL > 24*time.Hour {
		return Config{}, fmt.Errorf("AUTH_SESSION_TTL must be between 1 second and 24 hours")
	}
	originsValue := os.Getenv("CORS_ALLOWED_ORIGINS")
	if originsValue == "" {
		originsValue = "http://localhost:5173,http://127.0.0.1:5173"
	}
	var allowedOrigins []string
	for _, value := range strings.Split(originsValue, ",") {
		origin := strings.TrimSpace(value)
		u, err := url.Parse(origin)
		if err != nil || strings.Contains(origin, "*") || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.User != nil {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain comma-separated HTTP/HTTPS origins without paths")
		}
		allowedOrigins = append(allowedOrigins, origin)
	}

	return Config{
		Environment:            environment,
		Port:                   port,
		ShutdownTimeout:        shutdownTimeout,
		DatabaseURL:            databaseURL,
		DatabaseConnectTimeout: databaseConnectTimeout,
		MigrationsPath:         migrationsPath,
		AuthSessionTTL:         authSessionTTL,
		AllowedOrigins:         allowedOrigins,
	}, nil
}

func (c Config) Address() string {
	return fmt.Sprintf(":%d", c.Port)
}

func readPort(value string) (int, error) {
	if value == "" {
		return defaultPort, nil
	}

	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("PORT must be a number between 1 and 65535")
	}

	return port, nil
}

func readDuration(name, value string, fallback time.Duration) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}

	return duration, nil
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultEnvironment     = "development"
	defaultPort            = 8080
	defaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	Environment     string
	Port            int
	ShutdownTimeout time.Duration
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

	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = defaultEnvironment
	}

	return Config{
		Environment:     environment,
		Port:            port,
		ShutdownTimeout: shutdownTimeout,
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

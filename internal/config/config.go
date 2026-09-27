package config

import (
	"fmt"
	"os"
	"uuid"
)

type Environment string

const (
	EnvironmentLocal      Environment = "local"
	EnvironmentTest       Environment = "test"
	EnvironmentProduction Environment = "production"
)

type Config struct {
	Environment       Environment
	HTTPAddress       string
	DatabaseURL       string
	DevelopmentUserID uuid.UUID
}

func Load() (Config, error) {
	environment := Environment(getEnv("APP_ENV", string(EnvironmentLocal)))

	switch environment {
	case EnvironmentLocal, EnvironmentTest, EnvironmentProduction:
	default:
		return Config{}, fmt.Errorf("unsupported APP_ENV %q", environment)
	}

	developmentUserID, err := uuid.Parse(getEnv(
		"DEVELOPMENT_USER_ID",
		"01900000-0000-7000-8000-000000000001",
	))
	if err != nil {
		return Config{}, fmt.Errorf("parse DEVELOPMENT_USER_ID: %w", err)
	}

	return Config{
		Environment: environment,
		HTTPAddress: getEnv("HTTP_ADDRESS", ":8080"),
		DatabaseURL: getEnv(
			"DATABASE_URL",
			"postgres://tick_tack:tick_tack@localhost:5435/tick_tack?sslmode=disable",
		),
		DevelopmentUserID: developmentUserID,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}

	return fallback
}

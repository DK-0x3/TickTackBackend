package config

import (
	"testing"
	"uuid"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDRESS", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DEVELOPMENT_USER_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Environment != EnvironmentLocal {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, EnvironmentLocal)
	}

	if cfg.HTTPAddress != ":8080" {
		t.Fatalf("HTTPAddress = %q, want %q", cfg.HTTPAddress, ":8080")
	}

	const expectedDatabaseURL = "postgres://tick_tack:tick_tack@localhost:5435/tick_tack?sslmode=disable"
	if cfg.DatabaseURL != expectedDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, expectedDatabaseURL)
	}

	expectedUserID := uuid.MustParse("01900000-0000-7000-8000-000000000001")
	if cfg.DevelopmentUserID != expectedUserID {
		t.Fatalf("DevelopmentUserID = %s, want %s", cfg.DevelopmentUserID, expectedUserID)
	}
}

func TestLoadRejectsInvalidDevelopmentUserID(t *testing.T) {
	t.Setenv("DEVELOPMENT_USER_ID", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error")
	}
}

func TestLoadRejectsUnsupportedEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "staging")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error")
	}
}

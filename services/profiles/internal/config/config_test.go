package config

import (
	"testing"

	"github.com/nikita-simankov/upstore/shared/validate"
)

// clearEnv unsets the variables Load reads, so the developer's shell cannot change results.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "")
	t.Setenv("RABBITMQ_URL", "")
}

// TestLoadDefaults tests that Load succeeds with no environment variables set.
func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	if _, err := Load(); err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
}

// TestLoadRejectsInvalidURLs tests that invalid connection strings are reported together.
func TestLoadRejectsInvalidURLs(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "mysql://localhost/app")
	t.Setenv("RABBITMQ_URL", "http://broker")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}
	errs, ok := err.(validate.ValidationErrors)
	if !ok {
		t.Fatalf("error type = %T, want validate.ValidationErrors", err)
	}
	if len(errs) != 2 {
		t.Fatalf("got %d errors, want 2: %v", len(errs), errs)
	}
}

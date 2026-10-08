package config

import (
	"testing"

	"github.com/nikita-simankov/upstore/shared/validate"
)

// clearEnv unsets every variable Load reads so the developer's shell cannot change test results.
// env.Env treats an empty value as unset, so setting "" is enough.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HTTP_PORT", "GRPC_PORT", "LOG_LEVEL", "REDIS_URL",
		"ENVIRONMENT", "RABBITMQ_URL", "DATABASE_URL",
	} {
		t.Setenv(key, "")
	}
}

// TestLoadDefaults tests that Load succeeds with no environment variables set.
func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", cfg.HTTPPort)
	}
	if cfg.GRPCPort != 50051 {
		t.Errorf("GRPCPort = %d, want 50051", cfg.GRPCPort)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "development")
	}
}

// TestLoadPorts tests port parsing and range checks.
func TestLoadPorts(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		wantErr  bool
		wantPort int
	}{
		{"valid HTTP port", "HTTP_PORT", "9091", false, 9091},
		{"valid GRPC port", "GRPC_PORT", "50052", false, 50052},
		{"lowest valid port", "HTTP_PORT", "1", false, 1},
		{"highest valid port", "HTTP_PORT", "65535", false, 65535},
		{"not a number", "HTTP_PORT", "abc", true, 0},
		{"leading colon is rejected", "HTTP_PORT", ":9091", true, 0},
		{"zero", "HTTP_PORT", "0", true, 0},
		{"above range", "GRPC_PORT", "65536", true, 0},
		{"negative", "GRPC_PORT", "-1", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tt.key, tt.value)

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			got := cfg.HTTPPort
			if tt.key == "GRPC_PORT" {
				got = cfg.GRPCPort
			}
			if got != tt.wantPort {
				t.Errorf("%s = %d, want %d", tt.key, got, tt.wantPort)
			}
		})
	}
}

// TestLoadStringFields tests the string validators for the non-port fields.
func TestLoadStringFields(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{"valid log level", "LOG_LEVEL", "info", false},
		{"unknown log level", "LOG_LEVEL", "trace", true},
		{"valid environment", "ENVIRONMENT", "production", false},
		{"unknown environment", "ENVIRONMENT", "prod", true},
		{"valid redis URL", "REDIS_URL", "rediss://cache:6380", false},
		{"invalid redis URL", "REDIS_URL", "http://cache:6379", true},
		{"valid rabbitmq URL", "RABBITMQ_URL", "amqps://user:pass@mq:5671/", false},
		{"invalid rabbitmq URL", "RABBITMQ_URL", "amqp://mq:5672/", true},
		{"valid database URL", "DATABASE_URL", "postgresql://user:pass@db:5432/app", false},
		{"invalid database URL", "DATABASE_URL", "postgres://db:5432/app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLoadCollectsAllErrors tests that Load reports every invalid field, not just the first.
func TestLoadCollectsAllErrors(t *testing.T) {
	clearEnv(t)
	t.Setenv("HTTP_PORT", "abc")
	t.Setenv("LOG_LEVEL", "trace")

	cfg, err := Load()
	if err == nil {
		t.Fatal("Load() expected error, got nil")
	}
	if cfg != nil {
		t.Errorf("Load() returned config %+v alongside an error, want nil", cfg)
	}

	errs, ok := err.(validate.ValidationErrors)
	if !ok {
		t.Fatalf("error type = %T, want validate.ValidationErrors", err)
	}
	if len(errs) != 2 {
		t.Fatalf("got %d errors, want 2: %v", len(errs), errs)
	}
	if errs[0].Field != "HTTP_PORT" {
		t.Errorf("first error field = %q, want HTTP_PORT", errs[0].Field)
	}
	if errs[1].Field != "LOG_LEVEL" {
		t.Errorf("second error field = %q, want LOG_LEVEL", errs[1].Field)
	}
}

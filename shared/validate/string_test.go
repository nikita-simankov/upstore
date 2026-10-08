package validate

import (
	"strings"
	"testing"
)

// TestStringRequired tests the Required() method and default required behavior.
func TestStringRequired(t *testing.T) {
	schema := String("email").Required()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "user@example.com", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStringOptional tests the Optional() method allowing empty strings.
func TestStringOptional(t *testing.T) {
	schema := String("bio").Max(100).Optional()

	err := schema.Validate("")
	if err != nil {
		t.Errorf("expected no error for empty optional field, got %v", err)
	}

	longString := "a"
	for i := 0; i < 200; i++ {
		longString += "a"
	}
	err = schema.Validate(longString)
	if err == nil {
		t.Errorf("expected error for value exceeding max length")
	}
}

// TestStringEmail tests Email() validation with various email formats.
func TestStringEmail(t *testing.T) {
	schema := String("email").Email()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"test.email+tag@example.co.uk", false},
		{"invalid", true},
		{"@example.com", true},
		{"user@", true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Email validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringMinMax tests Min() and Max() length constraints.
func TestStringMinMax(t *testing.T) {
	schema := String("name").Min(2).Max(50)

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"ab", false},
		{"a", true},
		{"valid name", false},
		{strings.Repeat("x", 51), true}, // >50 chars
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Min/Max validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringLength tests exact length constraint.
func TestStringLength(t *testing.T) {
	schema := String("code").Length(6)

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"ABC123", false},
		{"ABCD", true},
		{"ABCDEFG", true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Length validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringRegex tests regex pattern matching.
func TestStringRegex(t *testing.T) {
	schema := String("username").Regex(`^[a-z0-9_]+$`)

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"valid_user123", false},
		{"ValidUser", true},
		{"user-name", true},
		{"user@name", true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Regex validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringOneOf tests OneOf() enum-like validation.
func TestStringOneOf(t *testing.T) {
	schema := String("env").OneOf("dev", "staging", "prod")

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"dev", false},
		{"staging", false},
		{"prod", false},
		{"test", true},
		{"", true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("OneOf validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringURL tests URL validation.
func TestStringURL(t *testing.T) {
	schema := String("website").URL()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"https://example.com", false},
		{"http://localhost:8080/path", false},
		{"example.com", true},
		{"ftp://invalid.com", false}, // Has scheme and host
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("URL validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringPostgresURL tests PostgreSQL URL validation with required credentials.
func TestStringPostgresURL(t *testing.T) {
	schema := String("db_url").PostgresURL()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"postgresql://user:pass@localhost:5432/mydb", false},
		{"postgres://user:pass@host/db", false},
		{"postgres://localhost/db", true},   // Missing user/password
		{"postgres://user@host/db", true},   // Missing password
		{"mysql://user:pass@host/db", true}, // Wrong scheme
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("PostgresURL validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringRedisURL tests Redis URL validation.
func TestStringRedisURL(t *testing.T) {
	schema := String("redis_url").RedisURL()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"redis://localhost:6379", false},
		{"rediss://localhost:6380", false},
		{"redis://", true},              // Missing host
		{"http://localhost:6379", true}, // Wrong scheme
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("RedisURL validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringRabbitMQURL tests RabbitMQ URL validation with credentials.
func TestStringRabbitMQURL(t *testing.T) {
	schema := String("rabbitmq_url").RabbitMQURL()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"amqp://user:pass@localhost:5672/", false},
		{"amqps://user:pass@rabbitmq:5671/", false},
		{"amqp://localhost:5672", true},  // Missing user/password
		{"amqp://user@host/", true},      // Missing password
		{"http://user:pass@host/", true}, // Wrong scheme
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("RabbitMQURL validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringRefine tests custom validation with Refine().
func TestStringRefine(t *testing.T) {
	schema := String("password").Min(8).Refine(
		func(s string) bool {
			hasUpper, hasLower, hasDigit := false, false, false
			for _, r := range s {
				if r >= 'A' && r <= 'Z' {
					hasUpper = true
				}
				if r >= 'a' && r <= 'z' {
					hasLower = true
				}
				if r >= '0' && r <= '9' {
					hasDigit = true
				}
			}
			return hasUpper && hasLower && hasDigit
		},
		"must contain uppercase, lowercase, and digit",
	)

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"ValidPass123", false},
		{"validpass123", true},
		{"VALIDPASS123", true},
		{"ValidPass", true},
		{"short", true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Refine validation for %q: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestStringValidateAll tests ValidateAll() collecting multiple errors.
func TestStringValidateAll(t *testing.T) {
	schema := String("username").Min(3).Max(20).Regex(`^[a-z0-9_]+$`)

	errs := schema.ValidateAll("ab")

	if len(errs) == 0 {
		t.Errorf("expected errors, got none")
	}

	if len(errs) < 1 {
		t.Errorf("expected at least 1 error, got %d", len(errs))
	}

	for _, err := range errs {
		if err.Field != "username" {
			t.Errorf("expected field 'username', got %q", err.Field)
		}
	}
}

// TestStringURLErrors covers the failure branches of URL validation.
func TestStringURLErrors(t *testing.T) {
	schema := String("website").URL()

	tests := []struct {
		name  string
		value string
	}{
		{"unparseable", "http://[::1"},
		{"missing scheme", "example.com"},
		{"missing host", "mailto:user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := schema.Validate(tt.value); err == nil {
				t.Errorf("Validate(%q) expected error, got nil", tt.value)
			}
		})
	}
}

// TestStringPostgresURLErrors covers the failure branches of PostgresURL validation.
func TestStringPostgresURLErrors(t *testing.T) {
	schema := String("db").PostgresURL()

	tests := []struct {
		name  string
		value string
	}{
		{"unparseable", "postgres://%zz@localhost/db"},
		{"wrong scheme", "mysql://user:pass@localhost/db"},
		{"missing host", "postgres:///db"},
		{"missing user", "postgres://localhost/db"},
		{"missing password", "postgres://user@localhost/db"},
		{"missing database", "postgres://user:pass@localhost/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := schema.Validate(tt.value); err == nil {
				t.Errorf("Validate(%q) expected error, got nil", tt.value)
			}
		})
	}
}

// TestStringRedisURLErrors covers the failure branches of RedisURL validation.
func TestStringRedisURLErrors(t *testing.T) {
	schema := String("redis").RedisURL()

	tests := []struct {
		name  string
		value string
	}{
		{"unparseable", "redis://%zz"},
		{"wrong scheme", "http://localhost:6379"},
		{"missing host", "redis:///0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := schema.Validate(tt.value); err == nil {
				t.Errorf("Validate(%q) expected error, got nil", tt.value)
			}
		})
	}
}

// TestStringRabbitMQURLErrors covers the failure branches of RabbitMQURL validation.
func TestStringRabbitMQURLErrors(t *testing.T) {
	schema := String("mq").RabbitMQURL()

	tests := []struct {
		name  string
		value string
	}{
		{"unparseable", "amqp://%zz"},
		{"wrong scheme", "http://user:pass@localhost:5672/"},
		{"missing host", "amqp:///"},
		{"missing user", "amqp://localhost:5672/"},
		{"missing password", "amqp://user@localhost:5672/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := schema.Validate(tt.value); err == nil {
				t.Errorf("Validate(%q) expected error, got nil", tt.value)
			}
		})
	}
}

package validate

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// StringSchema validates string values with support for length constraints,
// pattern matching, URL validation, and custom validation functions.
// StringSchema uses a fluent builder pattern for composing validators.
type StringSchema struct {
	field    string
	rules    []Rule[string]
	optional bool
}

// String creates a new StringSchema for validating a string field.
// The field parameter is used in error messages to identify which field failed validation.
// By default, the schema is required (empty strings fail validation).
//
// Example:
//
//	schema := String("email").Email().Required()
//	err := schema.Validate("user@example.com")
func String(field string) *StringSchema {
	return &StringSchema{
		field:    field,
		rules:    make([]Rule[string], 0),
		optional: false,
	}
}

// Optional marks the string field as optional, allowing empty values to pass validation.
// When a field is optional and the value is empty, all other validators are skipped.
// This is useful for fields like "description" or "nickname" that users can leave blank.
//
// Example:
//
//	schema := String("bio").Optional().Max(500)
//	schema.Validate("") // passes, field is optional
func (s *StringSchema) Optional() *StringSchema {
	s.optional = true
	return s
}

// Required marks the string field as required, rejecting empty values.
// This is the default behavior. Empty strings (including whitespace-only strings) fail validation.
// This method is useful when you want to explicitly enforce required fields in documentation.
//
// Example:
//
//	schema := String("username").Required().Min(3)
func (s *StringSchema) Required() *StringSchema {
	s.optional = false
	return s
}

// Min enforces a minimum string length (in UTF-8 runes, not bytes).
// Strings shorter than the specified length fail validation.
// This is UTF-8 aware, so multi-byte characters (emoji, accents, etc.) are counted correctly.
//
// Example:
//
//	schema := String("password").Min(8)
//	schema.Validate("short")   // fails: must be at least 8 characters
//	schema.Validate("goodpass") // passes
func (s *StringSchema) Min(length int) *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		if utf8.RuneCountInString(value) < length {
			return fmt.Errorf("must be at least %d characters", length)
		}
		return nil
	})
	return s
}

// Max enforces a maximum string length (in UTF-8 runes, not bytes).
// Strings longer than the specified length fail validation.
// This is UTF-8 aware, so multi-byte characters are counted correctly.
//
// Example:
//
//	schema := String("bio").Max(500)
//	schema.Validate(strings.Repeat("a", 501)) // fails: must be at most 500 characters
func (s *StringSchema) Max(length int) *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		if utf8.RuneCountInString(value) > length {
			return fmt.Errorf("must be at most %d characters", length)
		}
		return nil
	})
	return s
}

// Length enforces an exact string length (in UTF-8 runes, not bytes).
// Strings that are not exactly the specified length fail validation.
// This is UTF-8 aware, so multi-byte characters are counted correctly.
//
// Example:
//
//	schema := String("code").Length(6)
//	schema.Validate("ABC123") // passes
//	schema.Validate("ABCD")   // fails: must be exactly 6 characters
func (s *StringSchema) Length(length int) *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		if utf8.RuneCountInString(value) != length {
			return fmt.Errorf("must be exactly %d characters", length)
		}
		return nil
	})
	return s
}

// OneOf enforces that the string is one of a specified set of allowed values.
// This is case-sensitive. Empty values fail validation unless the field is optional.
//
// Example:
//
//	schema := String("env").OneOf("dev", "staging", "prod")
//	schema.Validate("dev") // passes
//	schema.Validate("test") // fails: must be one of [dev, staging, prod]
func (s *StringSchema) OneOf(values ...string) *StringSchema {
	allowed := make([]string, len(values))
	copy(allowed, values)

	s.rules = append(s.rules, func(value string) error {
		for _, a := range allowed {
			if value == a {
				return nil
			}
		}
		return fmt.Errorf("must be one of [%s]", strings.Join(allowed, ", "))
	})
	return s
}

// Email validates that the string is a valid email address according to RFC 5322.
// This uses Go's net.mail package for email parsing and validation.
//
// Example:
//
//	schema := String("contact").Email()
//	schema.Validate("user@example.com") // passes
//	schema.Validate("invalid-email")    // fails: invalid email address
func (s *StringSchema) Email() *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		_, err := mail.ParseAddress(value)
		if err != nil {
			return fmt.Errorf("invalid email address")
		}
		return nil
	})
	return s
}

// URL validates that the string is a valid URL with a scheme and host.
// The URL must include a scheme (e.g., "https://") and a host name.
//
// Example:
//
//	schema := String("website").URL()
//	schema.Validate("https://example.com")  // passes
//	schema.Validate("example.com")          // fails: must include scheme
func (s *StringSchema) URL() *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		u, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid URL: %v", err)
		}
		if u.Scheme == "" {
			return fmt.Errorf("URL must include a scheme (e.g. https://)")
		}
		if u.Host == "" {
			return fmt.Errorf("URL must include a host")
		}
		return nil
	})
	return s
}

// Regex validates that the string matches a regular expression pattern.
// The pattern is compiled once during validation, so invalid patterns will cause a panic.
// For complex patterns, consider compiling the regex yourself and using Refine() instead.
//
// Example:
//
//	schema := String("username").Regex(`^[a-z0-9_]+$`)
//	schema.Validate("valid_user123") // passes
//	schema.Validate("Invalid-User")  // fails: must match pattern
func (s *StringSchema) Regex(pattern string) *StringSchema {
	re := regexp.MustCompile(pattern)
	s.rules = append(s.rules, func(value string) error {
		if !re.MatchString(value) {
			return fmt.Errorf("must match pattern %s", pattern)
		}
		return nil
	})
	return s
}

// PostgresURL validates that the string is a valid PostgreSQL connection URL.
// It requires:
//   - Scheme: "postgres" or "postgresql"
//   - User: must be present with a username
//   - Password: must be present
//   - Host: must be present
//   - Database: must be specified in the path
//
// Example:
//
//	schema := String("db_url").PostgresURL()
//	schema.Validate("postgresql://user:pass@localhost:5432/mydb") // passes
//	schema.Validate("postgres://localhost/mydb")                   // fails: user/password required
func (s *StringSchema) PostgresURL() *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		u, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid postgres URL: %v", err)
		}
		if u.Scheme != "postgres" && u.Scheme != "postgresql" {
			return fmt.Errorf("scheme must be postgres:// or postgresql://, got %q", u.Scheme)
		}
		if u.Host == "" {
			return fmt.Errorf("host is required")
		}
		if u.User == nil {
			return fmt.Errorf("user is required")
		}
		if _, hasPassword := u.User.Password(); !hasPassword {
			return fmt.Errorf("password is required")
		}
		if strings.TrimPrefix(u.Path, "/") == "" {
			return fmt.Errorf("database name is required")
		}
		return nil
	})
	return s
}

// RedisURL validates that the string is a valid Redis connection URL.
// It requires:
//   - Scheme: "redis" or "rediss" (secure)
//   - Host: must be present
//
// Example:
//
//	schema := String("redis").RedisURL()
//	schema.Validate("redis://localhost:6379")  // passes
//	schema.Validate("rediss://localhost:6380") // passes (secure)
func (s *StringSchema) RedisURL() *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		u, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid redis URL: %v", err)
		}

		if u.Scheme != "redis" && u.Scheme != "rediss" {
			return fmt.Errorf("scheme must be redis:// or rediss://, got %q", u.Scheme)
		}

		if u.Host == "" {
			return fmt.Errorf("host is required")
		}

		return nil
	})
	return s
}

// RabbitMQURL validates that the string is a valid RabbitMQ (AMQP) connection URL.
// It requires:
//   - Scheme: "amqp" or "amqps" (secure)
//   - User: must be present with a username
//   - Password: must be present
//   - Host: must be present
//
// Example:
//
//	schema := String("mq_url").RabbitMQURL()
//	schema.Validate("amqp://user:pass@localhost:5672/")  // passes
//	schema.Validate("amqps://user:pass@rabbitmq:5671/") // passes (secure)
func (s *StringSchema) RabbitMQURL() *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		u, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid rabbitmq URL: %v", err)
		}
		if u.Scheme != "amqp" && u.Scheme != "amqps" {
			return fmt.Errorf("scheme must be amqp:// or amqps://, got %q", u.Scheme)
		}
		if u.Host == "" {
			return fmt.Errorf("host is required")
		}
		if u.User == nil {
			return fmt.Errorf("user is required")
		}
		if _, hasPassword := u.User.Password(); !hasPassword {
			return fmt.Errorf("password is required")
		}

		return nil
	})
	return s
}

// Refine adds a custom validation function to the schema.
// The function receives the string value and should return true if the value is valid,
// false otherwise. If validation fails, the provided message is used as the error.
// This is useful for complex validation logic that doesn't fit built-in validators.
//
// Example:
//
//	schema := String("password").Min(8).Refine(
//		func(s string) bool {
//			return hasUppercase(s) && hasLowercase(s) && hasDigit(s)
//		},
//		"must contain uppercase, lowercase, and digits",
//	)
func (s *StringSchema) Refine(fn func(string) bool, message string) *StringSchema {
	s.rules = append(s.rules, func(value string) error {
		if !fn(value) {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
	return s
}

// Validate runs all validation rules against the value and returns the first error.
// If validation passes, nil is returned. This method stops at the first error,
// making it suitable for fail-fast scenarios where you want immediate feedback.
//
// Example:
//
//	schema := String("email").Email().Required()
//	if err := schema.Validate(userInput); err != nil {
//		log.Printf("Invalid email: %v", err)
//	}
func (s *StringSchema) Validate(value string) error {
	errs := s.ValidateAll(value)
	if errs.HasErrors() {
		return errs[0]
	}
	return nil
}

// ValidateAll runs all validation rules against the value and returns all errors.
// If validation passes, an empty ValidationErrors slice is returned.
// This method collects all errors, making it suitable for scenarios where you want
// to show users all validation problems at once (e.g., in web forms).
//
// Example:
//
//	schema := String("username").Min(3).Max(20).Regex(`^[a-z0-9_]+$`)
//	errs := schema.ValidateAll(userInput)
//	if errs.HasErrors() {
//		for _, err := range errs {
//			fmt.Printf("%s: %s\n", err.Field, err.Message)
//		}
//	}
func (s *StringSchema) ValidateAll(value string) ValidationErrors {
	var errs ValidationErrors

	if strings.TrimSpace(value) == "" && !s.optional {
		return ValidationErrors{
			NewValidationError(s.field, "is required", "required"),
		}
	}

	if strings.TrimSpace(value) == "" && s.optional {
		return errs
	}

	for _, rule := range s.rules {
		if err := rule(value); err != nil {
			errs = append(errs, NewValidationError(s.field, err.Error(), "validation_failed"))
		}
	}

	return errs
}

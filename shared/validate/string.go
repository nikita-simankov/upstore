package validate

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"
)

type StringRule func(value string) error

type StringSchema struct {
	key   string
	rules []StringRule
}

func String(key string) StringSchema {
	return StringSchema{
		key:   key,
		rules: make([]StringRule, 0),
	}
}

func (s StringSchema) withRule(rule StringRule) StringSchema {
	rules := make([]StringRule, len(s.rules), len(s.rules)+1)

	copy(rules, s.rules)
	rules = append(rules, rule)

	return StringSchema{key: s.key, rules: rules}
}

func (s StringSchema) Min(length int) StringSchema {
	return s.withRule(func(value string) error {
		if utf8.RuneCountInString(value) < length {
			return fmt.Errorf("must be at least %d characters", length)
		}

		return nil
	})
}

func (s StringSchema) Max(length int) StringSchema {
	return s.withRule(func(value string) error {
		if utf8.RuneCountInString(value) > length {
			return fmt.Errorf("must be at most %d characters", length)
		}

		return nil
	})
}

func (s StringSchema) OneOf(allowed ...string) StringSchema {
	return s.withRule(func(value string) error {
		if value == "" {
			return nil
		}

		for _, a := range allowed {
			if value == a {
				return nil
			}
		}

		return fmt.Errorf("must be one of [%s], got %q", strings.Join(allowed, ", "), value)
	})
}

func (s StringSchema) Refine(fn func(string) bool, errorMessage string) StringSchema {
	return s.withRule(func(value string) error {
		if !fn(value) {
			return fmt.Errorf("%s", errorMessage)
		}

		return nil
	})
}

func (s StringSchema) Required() StringSchema {
	return s.withRule(func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("is required")
		}

		return nil
	})
}

func (s StringSchema) URL() StringSchema {
	return s.withRule(func(value string) error {
		if value == "" {
			return nil
		}

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
}

// PostgresURL — postgres:// или postgresql://, с user:password@host.
func (s StringSchema) PostgresURL() StringSchema {
	return s.withRule(func(value string) error {
		if value == "" {
			return nil
		}

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
}

func (s StringSchema) RedisURL() StringSchema {
	return s.withRule(func(value string) error {
		if value == "" {
			return nil
		}

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
}

func (s StringSchema) RabbitMQURL() StringSchema {
	return s.withRule(func(value string) error {
		if value == "" {
			return nil
		}

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
}

func (s StringSchema) Validate(value string) string {
	var errs []error

	for _, rule := range s.rules {
		if err := rule(value); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Printf("\t%s: %s", s.key, err)
		}
		os.Exit(1)
	}

	return value
}

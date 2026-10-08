package config

import (
	"github.com/nikita-simankov/upstore/shared/env"
	"github.com/nikita-simankov/upstore/shared/validate"
)

type Config struct {
	HTTPPort    int
	GRPCPort    int
	LogLevel    string
	RedisURL    string
	Environment string
	RabbitMQURL string
	DatabaseURL string
	// AccessSigningKey is the base64 Ed25519 seed that signs access tokens. It has no default:
	// a missing key must stop the service, not fall back to a key that is known to everyone.
	AccessSigningKey string
	// PublicAppURL is the front-end origin. Verification links point at its /verify-email page.
	PublicAppURL string
	// ResendAPIKey and MailFrom configure the Resend mailer. They are required outside development.
	// The key is a secret: it is never logged.
	ResendAPIKey string
	MailFrom     string
}

func Load() (*Config, error) {
	cfg := &Config{
		LogLevel:         env.Env("LOG_LEVEL", "debug"),
		RedisURL:         env.Env("REDIS_URL", "redis://valkey:6379"),
		Environment:      env.Env("ENVIRONMENT", "development"),
		RabbitMQURL:      env.Env("RABBITMQ_URL", "amqp://admin:password@rabbitmq:5672/"),
		DatabaseURL:      env.Env("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable"),
		AccessSigningKey: env.Env("ACCESS_SIGNING_KEY", ""),
		PublicAppURL:     env.Env("PUBLIC_APP_URL", "http://localhost:3000"),
		ResendAPIKey:     env.Env("RESEND_API_KEY", ""),
		MailFrom:         env.Env("MAIL_FROM", ""),
	}

	errs := validate.ValidateStruct(
		portField("HTTP_PORT", "9090", &cfg.HTTPPort),
		portField("GRPC_PORT", "50051", &cfg.GRPCPort),
		func() validate.ValidationErrors {
			return validate.String("ACCESS_SIGNING_KEY").Required().ValidateAll(cfg.AccessSigningKey)
		},
		func() validate.ValidationErrors {
			return validate.String("PUBLIC_APP_URL").Required().ValidateAll(cfg.PublicAppURL)
		},
		func() validate.ValidationErrors {
			return validate.String("LOG_LEVEL").OneOf("debug", "info", "warn", "error").ValidateAll(cfg.LogLevel)
		},
		func() validate.ValidationErrors {
			return validate.String("REDIS_URL").RedisURL().ValidateAll(cfg.RedisURL)
		},
		func() validate.ValidationErrors {
			return validate.String("ENVIRONMENT").OneOf("development", "production", "staging").ValidateAll(cfg.Environment)
		},
		func() validate.ValidationErrors {
			return validate.String("RABBITMQ_URL").RabbitMQURL().ValidateAll(cfg.RabbitMQURL)
		},
		func() validate.ValidationErrors {
			return validate.String("DATABASE_URL").PostgresURL().ValidateAll(cfg.DatabaseURL)
		},
	)
	// Outside development, a real mail provider is required, so the mail settings are mandatory.
	if cfg.Environment != "development" {
		if cfg.ResendAPIKey == "" {
			errs = append(errs, validate.NewValidationError("RESEND_API_KEY", "is required outside development", "required"))
		}
		if cfg.MailFrom == "" {
			errs = append(errs, validate.NewValidationError("MAIL_FROM", "is required outside development", "required"))
		}
	}

	if errs.HasErrors() {
		return nil, errs
	}

	return cfg, nil
}

// portField reads a port from the environment, parses it as an integer, and
// stores the result in dst. Values that are not integers or fall outside
// 1-65535 are reported as validation errors instead of being silently replaced
// by the fallback.
func portField(name, fallback string, dst *int) validate.ValidatorFunc {
	return func() validate.ValidationErrors {
		port, err := validate.Int(name).ParseString(env.Env(name, fallback))
		if err != nil {
			return validate.ValidationErrors{
				validate.NewValidationError(name, err.Error(), "parse"),
			}
		}

		*dst = port
		return validate.Int(name).Min(1).Max(65535).ValidateAll(port)
	}
}

package config

import (
	"github.com/nikita-simankov/upstore/shared/env"
	"github.com/nikita-simankov/upstore/shared/validate"
)

// Config holds the settings the profiles service needs to start.
type Config struct {
	DatabaseURL string
	RabbitMQURL string
}

// Load reads the configuration from the environment and validates it.
// It returns every invalid field, not just the first.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: env.Env("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable"),
		RabbitMQURL: env.Env("RABBITMQ_URL", "amqp://admin:password@rabbitmq:5672/"),
	}

	errs := validate.ValidateStruct(
		func() validate.ValidationErrors {
			return validate.String("DATABASE_URL").PostgresURL().ValidateAll(cfg.DatabaseURL)
		},
		func() validate.ValidationErrors {
			return validate.String("RABBITMQ_URL").RabbitMQURL().ValidateAll(cfg.RabbitMQURL)
		},
	)
	if errs.HasErrors() {
		return nil, errs
	}
	return cfg, nil
}

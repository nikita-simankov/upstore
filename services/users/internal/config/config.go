package config

import (
	"strings"

	"github.com/nikita-simankov/upstore/shared/env"
	"github.com/nikita-simankov/upstore/shared/validate"
)

type Config struct {
	HTTPPort    string
	GRPCPort    string
	LogLevel    string
	RedisURL    string
	Environment string
	RabbitMQURL string
	DatabaseURL string
}

func Load() *Config {
	cfg := &Config{
		HTTPPort: validate.
			String("HTTP_PORT").
			Required().
			Refine(func(port string) bool {
				return strings.HasPrefix(port, ":")
			}, "must start with ':'").
			Validate(env.Env("HTTP_PORT", ":9090")),
		GRPCPort: validate.
			String("GRPC_PORT").
			Required().
			Refine(func(port string) bool {
				return strings.HasPrefix(port, ":")
			}, "must start with ':'").
			Validate(env.Env("GRPC_PORT", ":50051")),
		LogLevel: validate.
			String("LOG_LEVEL").
			Required().
			OneOf("debug", "info", "warn", "error").
			Validate(env.Env("LOG_LEVEL", "debug")),
		RedisURL: validate.
			String("REDIS_URL").
			Required().
			RedisURL().
			Validate(env.Env("REDIS_URL", "redis://valkey:6379")),
		Environment: validate.
			String("ENVIRONMENT").
			Required().
			OneOf("development", "production", "staging").
			Validate(env.Env("ENVIRONMENT", "development")),
		RabbitMQURL: validate.
			String("RABBITMQ_URL").
			Required().
			RabbitMQURL().
			Validate(env.Env("RABBITMQ_URL", "amqp://admin:password@rabbitmq:5672/")),
		DatabaseURL: validate.
			String("DATABASE_URL").
			Required().
			PostgresURL().
			Validate(env.Env("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable")),
	}

	return cfg
}

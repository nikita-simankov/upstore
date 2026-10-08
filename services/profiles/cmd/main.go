package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/nikita-simankov/upstore/services/profiles/internal/config"
	"github.com/nikita-simankov/upstore/services/profiles/internal/profiles"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func init() {
	// .env is optional. In containers, configuration comes from the environment.
	_ = godotenv.Load()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run consumes account.created events until the process receives SIGINT or SIGTERM.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("rabbitmq: %w", err)
	}
	defer conn.Close()

	log.Printf("profiles %s consuming %s", version, profiles.QueueAccountCreated)
	return profiles.Consume(ctx, conn, pool)
}

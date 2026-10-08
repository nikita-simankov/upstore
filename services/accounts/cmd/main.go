package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/nikita-simankov/upstore/services/accounts/internal/config"
	"github.com/nikita-simankov/upstore/services/accounts/internal/outbox"
	"github.com/nikita-simankov/upstore/shared/events"
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

// run starts the outbox relay and the HTTP health endpoint, and blocks until the
// process receives SIGINT or SIGTERM.
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

	publisher, err := outbox.NewRabbitMQPublisher(conn, events.Exchange)
	if err != nil {
		return fmt.Errorf("rabbitmq publisher: %w", err)
	}
	defer publisher.Close()

	go outbox.NewRelay(pool, publisher, 100).Run(ctx, time.Second)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: healthHandler(pool),
	}
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()
	log.Printf("accounts %s listening on :%d", version, cfg.HTTPPort)

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// healthHandler serves GET /health. It returns 200 when the database answers, 503 otherwise.
func healthHandler(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

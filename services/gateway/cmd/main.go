package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/nikita-simankov/upstore/services/gateway/internal/config"
	"github.com/nikita-simankov/upstore/services/gateway/internal/gateway"
	"github.com/nikita-simankov/upstore/shared/authtoken"
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

// run serves the gateway until the process receives SIGINT or SIGTERM.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	publicKey, err := authtoken.ParsePublicKey(cfg.AccessPublicKey)
	if err != nil {
		return fmt.Errorf("access public key: %w", err)
	}
	accounts, err := url.Parse(cfg.AccountsURL)
	if err != nil {
		return fmt.Errorf("accounts url: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           gateway.New(publicKey, accounts).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()
	log.Printf("gateway %s listening on :%d", version, cfg.HTTPPort)

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

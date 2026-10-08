package config

import (
	"github.com/nikita-simankov/upstore/shared/env"
	"github.com/nikita-simankov/upstore/shared/validate"
)

// Config holds what the gateway needs. It never holds the signing key, only the public key.
type Config struct {
	HTTPPort int
	// AccessPublicKey is the base64 Ed25519 public key that verifies access tokens.
	AccessPublicKey string
	// AccountsURL is where auth requests are forwarded, for example http://accounts:9090.
	AccountsURL string
}

// Load reads the configuration from the environment and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		AccessPublicKey: env.Env("ACCESS_PUBLIC_KEY", ""),
		AccountsURL:     env.Env("ACCOUNTS_URL", "http://accounts:9090"),
	}

	port, err := validate.Int("HTTP_PORT").ParseString(env.Env("HTTP_PORT", "9090"))
	var errs validate.ValidationErrors
	if err != nil {
		errs = append(errs, validate.NewValidationError("HTTP_PORT", err.Error(), "parse"))
	} else {
		cfg.HTTPPort = port
		errs = append(errs, validate.Int("HTTP_PORT").Min(1).Max(65535).ValidateAll(port)...)
	}

	errs = append(errs, validate.String("ACCESS_PUBLIC_KEY").Required().ValidateAll(cfg.AccessPublicKey)...)
	errs = append(errs, validate.String("ACCOUNTS_URL").URL().Required().ValidateAll(cfg.AccountsURL)...)

	if errs.HasErrors() {
		return nil, errs
	}
	return cfg, nil
}

package config

import (
	"testing"

	"github.com/nikita-simankov/upstore/shared/validate"
)

// TestLoadRequiresPublicKeyAndAccountsURL tests that the gateway will not start without the key
// that verifies tokens, and that it reports every invalid field together.
func TestLoadRequiresPublicKeyAndAccountsURL(t *testing.T) {
	t.Setenv("ACCESS_PUBLIC_KEY", "")
	t.Setenv("ACCOUNTS_URL", "not a url")
	t.Setenv("HTTP_PORT", "70000")

	_, err := Load()
	errs, ok := err.(validate.ValidationErrors)
	if !ok {
		t.Fatalf("error = %v, want validate.ValidationErrors", err)
	}
	if len(errs) < 3 {
		t.Errorf("got %d errors, want at least 3 (key, URL, port): %v", len(errs), errs)
	}
}

// TestLoadAcceptsValidSettings tests the defaults with a key set.
func TestLoadAcceptsValidSettings(t *testing.T) {
	t.Setenv("ACCESS_PUBLIC_KEY", "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=")
	t.Setenv("ACCOUNTS_URL", "")
	t.Setenv("HTTP_PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPPort != 9090 || cfg.AccountsURL != "http://accounts:9090" {
		t.Errorf("defaults = %+v", cfg)
	}
}

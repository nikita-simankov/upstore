package validate

import (
	"strings"
	"testing"
)

// TestValidateStruct tests that ValidateStruct collects errors from every validator in order.
func TestValidateStruct(t *testing.T) {
	t.Run("no validators", func(t *testing.T) {
		if errs := ValidateStruct(); errs.HasErrors() {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("all passing", func(t *testing.T) {
		errs := ValidateStruct(
			func() ValidationErrors { return String("email").Email().ValidateAll("user@example.com") },
			func() ValidationErrors { return Int("port").Min(1).ValidateAll(8080) },
		)
		if errs.HasErrors() {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("collects errors in order", func(t *testing.T) {
		errs := ValidateStruct(
			func() ValidationErrors { return String("email").Email().ValidateAll("bad") },
			func() ValidationErrors { return Int("port").Min(1).ValidateAll(0) },
		)
		if len(errs) != 2 {
			t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "email" {
			t.Errorf("first error field = %q, want %q", errs[0].Field, "email")
		}
		if errs[1].Field != "port" {
			t.Errorf("second error field = %q, want %q", errs[1].Field, "port")
		}
	})
}

// TestMustValidate tests that MustValidate is silent for nil and panics for an error.
func TestMustValidate(t *testing.T) {
	t.Run("nil does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		MustValidate(nil)
	})

	t.Run("error panics", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic, got none")
			}
			if msg, ok := r.(string); !ok || !strings.Contains(msg, "validation failed") {
				t.Errorf("unexpected panic value: %v", r)
			}
		}()
		MustValidate(String("name").Required().Validate(""))
	})
}

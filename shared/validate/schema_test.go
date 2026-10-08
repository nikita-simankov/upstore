package validate

import (
	"errors"
	"testing"
)

// TestBaseSchemaExecuteRules tests the shared rule runner used by schema types.
func TestBaseSchemaExecuteRules(t *testing.T) {
	isEmpty := func(s string) bool { return s == "" }

	t.Run("empty and required", func(t *testing.T) {
		b := &BaseSchema[string]{field: "name"}
		errs := b.executeRules("", isEmpty)
		if len(errs) != 1 || errs[0].Code != "required" {
			t.Errorf("expected one required error, got %v", errs)
		}
	})

	t.Run("empty and optional", func(t *testing.T) {
		b := &BaseSchema[string]{field: "name", optional: true}
		if errs := b.executeRules("", isEmpty); len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("failing rule", func(t *testing.T) {
		b := &BaseSchema[string]{field: "name"}
		b.addRule(func(s string) error { return errors.New("too short") })
		errs := b.executeRules("ab", isEmpty)
		if len(errs) != 1 {
			t.Fatalf("expected one error, got %v", errs)
		}
		if errs[0].Field != "name" || errs[0].Message != "too short" || errs[0].Code != "validation_failed" {
			t.Errorf("unexpected error: %+v", errs[0])
		}
	})

	t.Run("passing rules", func(t *testing.T) {
		b := &BaseSchema[string]{field: "name"}
		b.addRule(func(s string) error { return nil })
		if errs := b.executeRules("alice", isEmpty); len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})
}

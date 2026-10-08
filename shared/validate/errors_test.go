package validate

import (
	"testing"
)

// TestValidationErrorError tests ValidationError's Error() method.
func TestValidationErrorError(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Message: "invalid email address",
		Code:    "invalid_email",
	}

	expected := "email: invalid email address"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

// TestNewValidationError tests NewValidationError constructor.
func TestNewValidationError(t *testing.T) {
	err := NewValidationError("password", "too short", "min_length")

	if err.Field != "password" {
		t.Errorf("Field = %q, want %q", err.Field, "password")
	}
	if err.Message != "too short" {
		t.Errorf("Message = %q, want %q", err.Message, "too short")
	}
	if err.Code != "min_length" {
		t.Errorf("Code = %q, want %q", err.Code, "min_length")
	}
}

// TestValidationErrorsError tests ValidationErrors Error() method with multiple errors.
func TestValidationErrorsError(t *testing.T) {
	errs := ValidationErrors{
		NewValidationError("name", "is required", "required"),
		NewValidationError("email", "invalid format", "invalid_email"),
	}

	result := errs.Error()
	expected1 := "name: is required"
	expected2 := "email: invalid format"

	if result != expected1+"\n"+expected2 {
		t.Errorf("Error() = %q, want %q", result, expected1+"\n"+expected2)
	}
}

// TestValidationErrorsErrorEmpty tests ValidationErrors Error() with no errors.
func TestValidationErrorsErrorEmpty(t *testing.T) {
	errs := ValidationErrors{}

	expected := "validation passed"
	if errs.Error() != expected {
		t.Errorf("Error() = %q, want %q", errs.Error(), expected)
	}
}

// TestValidationErrorsHasErrors tests HasErrors() method.
func TestValidationErrorsHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		errs     ValidationErrors
		expected bool
	}{
		{
			name:     "with errors",
			errs:     ValidationErrors{NewValidationError("field", "error", "code")},
			expected: true,
		},
		{
			name:     "no errors",
			errs:     ValidationErrors{},
			expected: false,
		},
		{
			name:     "multiple errors",
			errs:     ValidationErrors{NewValidationError("a", "1", "x"), NewValidationError("b", "2", "y")},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errs.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestValidationErrorsErrorsByField tests ErrorsByField() filtering.
func TestValidationErrorsErrorsByField(t *testing.T) {
	errs := ValidationErrors{
		NewValidationError("name", "is required", "required"),
		NewValidationError("email", "invalid format", "invalid_email"),
		NewValidationError("name", "too short", "min_length"),
	}

	nameErrors := errs.ErrorsByField("name")
	if len(nameErrors) != 2 {
		t.Errorf("ErrorsByField('name') returned %d errors, want 2", len(nameErrors))
	}

	for _, err := range nameErrors {
		if err.Field != "name" {
			t.Errorf("ErrorsByField('name') returned error with Field=%q", err.Field)
		}
	}

	emailErrors := errs.ErrorsByField("email")
	if len(emailErrors) != 1 {
		t.Errorf("ErrorsByField('email') returned %d errors, want 1", len(emailErrors))
	}

	nonexistentErrors := errs.ErrorsByField("phone")
	if len(nonexistentErrors) != 0 {
		t.Errorf("ErrorsByField('phone') returned %d errors, want 0", len(nonexistentErrors))
	}
}

// TestValidationErrorsImplementsError tests that ValidationError implements error interface.
func TestValidationErrorImplementsError(t *testing.T) {
	var err error = NewValidationError("test", "message", "code")
	if err.Error() != "test: message" {
		t.Errorf("ValidationError doesn't properly implement error interface")
	}
}

// TestValidationErrorsImplementsError tests that ValidationErrors implements error interface.
func TestValidationErrorsImplementError(t *testing.T) {
	var err error = ValidationErrors{
		NewValidationError("field", "message", "code"),
	}

	result := err.Error()
	if result != "field: message" {
		t.Errorf("ValidationErrors Error() = %q, want %q", result, "field: message")
	}
}

// TestValidationErrorsErrorsByFieldEmpty tests ErrorsByField() with empty ValidationErrors.
func TestValidationErrorsErrorsByFieldEmpty(t *testing.T) {
	errs := ValidationErrors{}

	result := errs.ErrorsByField("anyfield")
	if result != nil && len(result) != 0 {
		t.Errorf("ErrorsByField() on empty ValidationErrors should return empty, got %v", result)
	}
}

// TestValidationErrorsCaseSensitive tests that ErrorsByField is case-sensitive.
func TestValidationErrorsCaseSensitive(t *testing.T) {
	errs := ValidationErrors{
		NewValidationError("Name", "error", "code"),
	}

	// Should not find due to case difference
	lowercase := errs.ErrorsByField("name")
	if len(lowercase) != 0 {
		t.Errorf("ErrorsByField() should be case-sensitive, found %d errors", len(lowercase))
	}

	// Should find with correct case
	correct := errs.ErrorsByField("Name")
	if len(correct) != 1 {
		t.Errorf("ErrorsByField() with correct case should find 1 error, got %d", len(correct))
	}
}

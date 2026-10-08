// Package validate provides a Zod-like validation library for Go with support for
// strings, numbers, arrays, and struct composition. It replaces the unsafe v1 library
// by returning structured errors instead of calling os.Exit(1).
package validate

import (
	"fmt"
	"strings"
)

// ValidationError represents a single validation failure for a field.
// It contains the field name, error message, and an error code for programmatic handling.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string
	// Message is the human-readable error message explaining why validation failed.
	Message string
	// Code is a machine-readable error code for programmatic handling (e.g., "required", "min_length").
	Code string
}

// Error implements the error interface for ValidationError, returning a formatted
// error message in the format "field: message".
func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidationErrors is a collection of validation errors, typically returned by
// schema ValidateAll() methods. An empty ValidationErrors means all validations passed.
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors, returning a formatted
// error message with all errors separated by newlines. If there are no errors,
// it returns "validation passed".
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation passed"
	}

	var sb strings.Builder
	for i, err := range ve {
		sb.WriteString(fmt.Sprintf("%s: %s", err.Field, err.Message))
		if i < len(ve)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// HasErrors returns true if the ValidationErrors collection contains one or more errors.
// This is useful for checking if validation passed without using error != nil.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// ErrorsByField returns all errors for a specific field name.
// This is useful when you want to handle errors for a particular field separately.
// If no errors exist for the field, an empty slice is returned.
func (ve ValidationErrors) ErrorsByField(field string) []ValidationError {
	var errs []ValidationError
	for _, err := range ve {
		if err.Field == field {
			errs = append(errs, err)
		}
	}
	return errs
}

// NewValidationError creates a new ValidationError with the given field, message, and code.
// This is typically used when building custom validators or wrapping validation results.
//
// Example:
//
//	err := NewValidationError("email", "invalid email format", "invalid_email")
func NewValidationError(field, message, code string) ValidationError {
	return ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	}
}

package validate

// Rule[T] is a validation function that checks a value and returns an error if validation fails.
// The error message should describe why the value is invalid.
//
// Example:
//
//	rule := func(value string) error {
//		if len(value) < 3 {
//			return fmt.Errorf("must be at least 3 characters")
//		}
//		return nil
//	}
type Rule[T any] func(value T) error

// Schema[T] defines the interface that all schema types must implement.
// It provides two methods for validation: Validate (returns first error) and
// ValidateAll (returns all errors).
type Schema[T any] interface {
	// Validate returns the first validation error, or nil if validation passes.
	Validate(value T) error
	// ValidateAll returns all validation errors, or empty slice if validation passes.
	ValidateAll(value T) ValidationErrors
}

// BaseSchema[T] provides common validation logic for generic schema types.
// It tracks the field name, validation rules, and whether the field is optional.
// This type is typically embedded in concrete schema implementations.
type BaseSchema[T any] struct {
	field    string
	rules    []Rule[T]
	optional bool
}

// addRule appends a validation rule to the schema's rule list.
// This is an internal helper method used by schema implementations.
func (b *BaseSchema[T]) addRule(rule Rule[T]) {
	b.rules = append(b.rules, rule)
}

// executeRules runs all validation rules against a value and collects errors.
// It handles optional field logic: if the field is optional and the value is empty,
// validation passes. Otherwise, all rules are executed and errors are collected.
//
// Parameters:
//   - value: The value to validate
//   - isEmpty: A function that determines if the value is considered empty for this type
//
// Returns ValidationErrors with all validation failures, or empty if validation passes.
func (b *BaseSchema[T]) executeRules(value T, isEmpty func(T) bool) ValidationErrors {
	var errs ValidationErrors

	if isEmpty(value) {
		if b.optional {
			return errs
		}
		return ValidationErrors{
			NewValidationError(b.field, "is required", "required"),
		}
	}

	for _, rule := range b.rules {
		if err := rule(value); err != nil {
			errs = append(errs, NewValidationError(b.field, err.Error(), "validation_failed"))
		}
	}

	return errs
}

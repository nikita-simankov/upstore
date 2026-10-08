package validate

import (
	"fmt"
)

// ValidatorFunc is a function that performs validation and returns any errors.
// It is used with ValidateStruct to compose multiple field validators.
// ValidatorFunc should call ValidateAll() on a schema and return the resulting errors.
//
// Example:
//
//	func() ValidationErrors {
//		return String("email").Email().Required().ValidateAll(config.Email)
//	}
type ValidatorFunc func() ValidationErrors

// ValidateStruct runs multiple validator functions and collects all errors in a single result.
// This is useful for validating structs or configuration objects where you want to validate
// all fields at once and report all errors together (rather than stopping at the first error).
// Validators run in the order provided.
//
// Example:
//
//	errors := ValidateStruct(
//		func() ValidationErrors {
//			return String("email").Email().Required().ValidateAll(config.Email)
//		},
//		func() ValidationErrors {
//			return Int("port").Min(1).Max(65535).ValidateAll(config.Port)
//		},
//		func() ValidationErrors {
//			return String("url").URL().Required().ValidateAll(config.URL)
//		},
//	)
//	if errors.HasErrors() {
//		for _, err := range errors {
//			log.Printf("%s: %s", err.Field, err.Message)
//		}
//	}
func ValidateStruct(validators ...ValidatorFunc) ValidationErrors {
	var allErrs ValidationErrors

	for _, validator := range validators {
		errs := validator()
		allErrs = append(allErrs, errs...)
	}

	return allErrs
}

// MustValidate panics if the provided error is not nil.
// This is useful in scenarios where validation failure is a fatal condition
// that should crash the program (e.g., during initialization).
// Prefer returning errors rather than using this function in production code.
//
// Example:
//
//	cfg, err := config.Load()
//	MustValidate(err) // Panics if config is invalid
//	// cfg is guaranteed to be valid if we reach here
func MustValidate(err error) {
	if err != nil {
		panic(fmt.Sprintf("validation failed: %v", err))
	}
}

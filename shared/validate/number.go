package validate

import (
	"fmt"
	"strconv"
)

// IntSchema validates integer values with support for range constraints and custom validation.
// IntSchema uses a fluent builder pattern for composing validators.
type IntSchema struct {
	field string
	rules []Rule[int]
}

// Int creates a new IntSchema for validating an integer field.
// The field parameter is used in error messages to identify which field failed validation.
//
// Example:
//
//	schema := Int("port").Min(1).Max(65535)
//	port, err := schema.ParseString("8080")
func Int(field string) *IntSchema {
	return &IntSchema{
		field: field,
		rules: make([]Rule[int], 0),
	}
}

// Min enforces a minimum value constraint.
// Values less than the specified minimum fail validation.
//
// Example:
//
//	schema := Int("count").Min(1)
//	schema.Validate(0)  // fails: must be at least 1
//	schema.Validate(10) // passes
func (i *IntSchema) Min(min int) *IntSchema {
	i.rules = append(i.rules, func(value int) error {
		if value < min {
			return fmt.Errorf("must be at least %d", min)
		}
		return nil
	})
	return i
}

// Max enforces a maximum value constraint.
// Values greater than the specified maximum fail validation.
//
// Example:
//
//	schema := Int("port").Max(65535)
//	schema.Validate(99999) // fails: must be at most 65535
//	schema.Validate(8080)  // passes
func (i *IntSchema) Max(max int) *IntSchema {
	i.rules = append(i.rules, func(value int) error {
		if value > max {
			return fmt.Errorf("must be at most %d", max)
		}
		return nil
	})
	return i
}

// Positive enforces that the value is greater than zero.
// This is equivalent to Min(1) but with a simpler error message.
//
// Example:
//
//	schema := Int("quantity").Positive()
//	schema.Validate(0)  // fails: must be positive
//	schema.Validate(-1) // fails: must be positive
//	schema.Validate(5)  // passes
func (i *IntSchema) Positive() *IntSchema {
	i.rules = append(i.rules, func(value int) error {
		if value <= 0 {
			return fmt.Errorf("must be positive")
		}
		return nil
	})
	return i
}

// Negative enforces that the value is less than zero.
// This is useful for validating offsets or negative adjustments.
//
// Example:
//
//	schema := Int("offset").Negative()
//	schema.Validate(-5) // passes
//	schema.Validate(5)  // fails: must be negative
func (i *IntSchema) Negative() *IntSchema {
	i.rules = append(i.rules, func(value int) error {
		if value >= 0 {
			return fmt.Errorf("must be negative")
		}
		return nil
	})
	return i
}

// OneOf enforces that the value is one of a specified set of allowed values.
// This is useful for enum-like integer validation.
//
// Example:
//
//	schema := Int("status").OneOf(0, 1, 2)
//	schema.Validate(1) // passes
//	schema.Validate(3) // fails: must be one of the allowed values
func (i *IntSchema) OneOf(allowed ...int) *IntSchema {
	i.rules = append(i.rules, func(value int) error {
		for _, a := range allowed {
			if value == a {
				return nil
			}
		}
		return fmt.Errorf("must be one of the allowed values")
	})
	return i
}

// Refine adds a custom validation function to the schema.
// The function receives the integer value and should return true if the value is valid,
// false otherwise. If validation fails, the provided message is used as the error.
// This is useful for complex validation logic that doesn't fit built-in validators.
//
// Example:
//
//	schema := Int("count").Refine(
//		func(val int) bool { return val%2 == 0 },
//		"must be an even number",
//	)
func (i *IntSchema) Refine(fn func(int) bool, message string) *IntSchema {
	i.rules = append(i.rules, func(value int) error {
		if !fn(value) {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
	return i
}

// Validate runs all validation rules against the value and returns the first error.
// If validation passes, nil is returned. This method stops at the first error,
// making it suitable for fail-fast scenarios.
//
// Example:
//
//	schema := Int("port").Min(1).Max(65535)
//	if err := schema.Validate(8080); err != nil {
//		log.Printf("Invalid port: %v", err)
//	}
func (i *IntSchema) Validate(value int) error {
	errs := i.ValidateAll(value)
	if errs.HasErrors() {
		return errs[0]
	}
	return nil
}

// ValidateAll runs all validation rules against the value and returns all errors.
// If validation passes, an empty ValidationErrors slice is returned.
// This method collects all errors, making it suitable for scenarios where you want
// to show all validation problems at once.
//
// Example:
//
//	schema := Int("port").Min(1).Max(65535).Refine(isAvailable, "port in use")
//	errs := schema.ValidateAll(port)
//	if errs.HasErrors() {
//		for _, err := range errs {
//			fmt.Printf("%s: %s\n", err.Field, err.Message)
//		}
//	}
func (i *IntSchema) ValidateAll(value int) ValidationErrors {
	var errs ValidationErrors

	for _, rule := range i.rules {
		if err := rule(value); err != nil {
			errs = append(errs, NewValidationError(i.field, err.Error(), "validation_failed"))
		}
	}

	return errs
}

// ParseString parses a string value into an integer and validates it.
// This is useful for converting environment variables or request parameters to validated integers.
// If parsing fails, a validation error is returned with a "parse" code.
// If parsing succeeds, all validation rules are applied.
//
// Example:
//
//	schema := Int("port").Min(1).Max(65535)
//	port, err := schema.ParseString(os.Getenv("PORT"))
//	if err != nil {
//		log.Fatalf("invalid port: %v", err)
//	}
func (i *IntSchema) ParseString(value string) (int, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %v", err)
	}
	return int(parsed), nil
}

// FloatSchema validates floating-point (decimal) values with support for range constraints
// and custom validation. FloatSchema uses a fluent builder pattern for composing validators.
type FloatSchema struct {
	field string
	rules []Rule[float64]
}

// Float creates a new FloatSchema for validating a floating-point field.
// The field parameter is used in error messages to identify which field failed validation.
//
// Example:
//
//	schema := Float("rate").Min(0.0).Max(100.0)
//	rate, err := schema.ParseString("95.5")
func Float(field string) *FloatSchema {
	return &FloatSchema{
		field: field,
		rules: make([]Rule[float64], 0),
	}
}

// Min enforces a minimum value constraint.
// Values less than the specified minimum fail validation.
//
// Example:
//
//	schema := Float("discount").Min(0.0)
//	schema.Validate(-0.1) // fails: must be at least 0
//	schema.Validate(0.5)  // passes
func (f *FloatSchema) Min(min float64) *FloatSchema {
	f.rules = append(f.rules, func(value float64) error {
		if value < min {
			return fmt.Errorf("must be at least %g", min)
		}
		return nil
	})
	return f
}

// Max enforces a maximum value constraint.
// Values greater than the specified maximum fail validation.
//
// Example:
//
//	schema := Float("rate").Max(100.0)
//	schema.Validate(150.0) // fails: must be at most 100
//	schema.Validate(95.5)  // passes
func (f *FloatSchema) Max(max float64) *FloatSchema {
	f.rules = append(f.rules, func(value float64) error {
		if value > max {
			return fmt.Errorf("must be at most %g", max)
		}
		return nil
	})
	return f
}

// Positive enforces that the value is greater than zero.
// This is equivalent to Min(0) but with a simpler error message.
// Note: This requires strictly positive (> 0), not non-negative (>= 0).
//
// Example:
//
//	schema := Float("price").Positive()
//	schema.Validate(0.0) // fails: must be positive
//	schema.Validate(9.99) // passes
func (f *FloatSchema) Positive() *FloatSchema {
	f.rules = append(f.rules, func(value float64) error {
		if value <= 0 {
			return fmt.Errorf("must be positive")
		}
		return nil
	})
	return f
}

// Negative enforces that the value is less than zero.
// This is useful for validating offsets or negative adjustments.
//
// Example:
//
//	schema := Float("adjustment").Negative()
//	schema.Validate(-5.5) // passes
//	schema.Validate(5.5)  // fails: must be negative
func (f *FloatSchema) Negative() *FloatSchema {
	f.rules = append(f.rules, func(value float64) error {
		if value >= 0 {
			return fmt.Errorf("must be negative")
		}
		return nil
	})
	return f
}

// Refine adds a custom validation function to the schema.
// The function receives the float value and should return true if the value is valid,
// false otherwise. If validation fails, the provided message is used as the error.
// This is useful for complex validation logic that doesn't fit built-in validators.
//
// Example:
//
//	schema := Float("tax_rate").Min(0.0).Max(1.0).Refine(
//		func(val float64) bool { return val*100 == float64(int(val*100)) },
//		"must have at most 2 decimal places",
//	)
func (f *FloatSchema) Refine(fn func(float64) bool, message string) *FloatSchema {
	f.rules = append(f.rules, func(value float64) error {
		if !fn(value) {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
	return f
}

// Validate runs all validation rules against the value and returns the first error.
// If validation passes, nil is returned. This method stops at the first error,
// making it suitable for fail-fast scenarios.
//
// Example:
//
//	schema := Float("rate").Min(0.0).Max(100.0)
//	if err := schema.Validate(95.5); err != nil {
//		log.Printf("Invalid rate: %v", err)
//	}
func (f *FloatSchema) Validate(value float64) error {
	errs := f.ValidateAll(value)
	if errs.HasErrors() {
		return errs[0]
	}
	return nil
}

// ValidateAll runs all validation rules against the value and returns all errors.
// If validation passes, an empty ValidationErrors slice is returned.
// This method collects all errors, making it suitable for scenarios where you want
// to show all validation problems at once.
//
// Example:
//
//	schema := Float("rate").Min(0.0).Max(100.0)
//	errs := schema.ValidateAll(rate)
//	if errs.HasErrors() {
//		for _, err := range errs {
//			fmt.Printf("%s: %s\n", err.Field, err.Message)
//		}
//	}
func (f *FloatSchema) ValidateAll(value float64) ValidationErrors {
	var errs ValidationErrors

	for _, rule := range f.rules {
		if err := rule(value); err != nil {
			errs = append(errs, NewValidationError(f.field, err.Error(), "validation_failed"))
		}
	}

	return errs
}

// ParseString parses a string value into a float and validates it.
// This is useful for converting environment variables or request parameters to validated floats.
// If parsing fails, a validation error is returned with a "parse" code.
// If parsing succeeds, all validation rules are applied.
//
// Example:
//
//	schema := Float("temperature").Min(-50.0).Max(50.0)
//	temp, err := schema.ParseString(userInput)
//	if err != nil {
//		log.Fatalf("invalid temperature: %v", err)
//	}
func (f *FloatSchema) ParseString(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float: %v", err)
	}
	return parsed, nil
}

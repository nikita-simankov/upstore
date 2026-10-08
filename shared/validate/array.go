package validate

import (
	"fmt"
)

// ArraySchema[T] validates arrays/slices of a specific type with support for length constraints
// and per-item validation. ArraySchema uses a fluent builder pattern for composing validators.
// The type parameter T specifies the type of elements in the array.
type ArraySchema[T any] struct {
	field      string
	rules      []Rule[[]T]
	optional   bool
	itemSchema Schema[T]
}

// Array creates a new ArraySchema for validating an array field.
// The field parameter is used in error messages to identify which field failed validation.
// The type parameter T specifies the type of elements in the array (e.g., Array[string]).
// By default, the schema is required (empty arrays fail validation unless explicitly marked optional).
//
// Example:
//
//	schema := Array[string]("tags").Min(1).Max(5)
//	errs := schema.ValidateAll([]string{"go", "backend", "api"})
func Array[T any](field string) *ArraySchema[T] {
	return &ArraySchema[T]{
		field:    field,
		rules:    make([]Rule[[]T], 0),
		optional: false,
	}
}

// Optional marks the array field as optional, allowing empty arrays to pass validation.
// When a field is optional and the array is empty, all other validators are skipped.
// This is useful for optional list fields where the user may not provide any items.
//
// Example:
//
//	schema := Array[string]("hobbies").Optional().Max(10)
func (a *ArraySchema[T]) Optional() *ArraySchema[T] {
	a.optional = true
	return a
}

// Required marks the array field as required, rejecting empty arrays.
// This is the default behavior. An empty array will fail validation with "is required" error.
// This method is useful when you want to explicitly enforce required fields in documentation.
//
// Example:
//
//	schema := Array[int]("ids").Required().Min(1)
func (a *ArraySchema[T]) Required() *ArraySchema[T] {
	a.optional = false
	return a
}

// Min enforces a minimum number of items in the array.
// Arrays with fewer items than specified fail validation.
//
// Example:
//
//	schema := Array[string]("tags").Min(1)
//	schema.Validate([]string{})  // fails: must have at least 1 items
//	schema.Validate([]string{"go"}) // passes
func (a *ArraySchema[T]) Min(length int) *ArraySchema[T] {
	a.rules = append(a.rules, func(value []T) error {
		if len(value) < length {
			return fmt.Errorf("must have at least %d items", length)
		}
		return nil
	})
	return a
}

// Max enforces a maximum number of items in the array.
// Arrays with more items than specified fail validation.
//
// Example:
//
//	schema := Array[string]("tags").Max(5)
//	schema.Validate([]string{"a", "b", "c", "d", "e", "f"}) // fails: must have at most 5 items
//	schema.Validate([]string{"a", "b", "c"}) // passes
func (a *ArraySchema[T]) Max(length int) *ArraySchema[T] {
	a.rules = append(a.rules, func(value []T) error {
		if len(value) > length {
			return fmt.Errorf("must have at most %d items", length)
		}
		return nil
	})
	return a
}

// Length enforces an exact number of items in the array.
// Arrays that don't have exactly the specified number of items fail validation.
//
// Example:
//
//	schema := Array[string]("coordinates").Length(2)
//	schema.Validate([]string{"10", "20"}) // passes
//	schema.Validate([]string{"10"})       // fails: must have exactly 2 items
func (a *ArraySchema[T]) Length(length int) *ArraySchema[T] {
	a.rules = append(a.rules, func(value []T) error {
		if len(value) != length {
			return fmt.Errorf("must have exactly %d items", length)
		}
		return nil
	})
	return a
}

// Items sets a schema to validate each item in the array.
// All items in the array must pass validation against the provided schema.
// If any item fails, that error is included with the array index in the field name.
//
// Example:
//
//	emailSchema := String("email").Email()
//	schema := Array[string]("recipients").Items(emailSchema)
//	errs := schema.ValidateAll([]string{"user@example.com", "invalid"})
//	// Returns error for "recipients[1]: invalid email address"
func (a *ArraySchema[T]) Items(schema Schema[T]) *ArraySchema[T] {
	a.itemSchema = schema
	return a
}

// Unique enforces that all items in the array are unique (no duplicates).
// Items are compared by their string representation using fmt.Sprintf("%v", item).
// If duplicates are found, validation fails with the index of the first duplicate.
//
// Example:
//
//	schema := Array[int]("ids").Unique()
//	schema.Validate([]int{1, 2, 3})     // passes
//	schema.Validate([]int{1, 2, 2, 3}) // fails: duplicate item at index 2
func (a *ArraySchema[T]) Unique() *ArraySchema[T] {
	a.rules = append(a.rules, func(value []T) error {
		seen := make(map[string]bool)
		for i, item := range value {
			key := fmt.Sprintf("%v", item)
			if seen[key] {
				return fmt.Errorf("duplicate item at index %d", i)
			}
			seen[key] = true
		}
		return nil
	})
	return a
}

// Refine adds a custom validation function to the schema.
// The function receives the entire array and should return true if the array is valid,
// false otherwise. If validation fails, the provided message is used as the error.
// This is useful for complex validation logic like checking array relationships or ordering.
//
// Example:
//
//	schema := Array[int]("prices").Refine(
//		func(prices []int) bool {
//			return prices[0] < prices[len(prices)-1] // ascending order
//		},
//		"prices must be in ascending order",
//	)
func (a *ArraySchema[T]) Refine(fn func([]T) bool, message string) *ArraySchema[T] {
	a.rules = append(a.rules, func(value []T) error {
		if !fn(value) {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
	return a
}

// Validate runs all validation rules against the array and returns the first error.
// If validation passes, nil is returned. This method stops at the first error,
// making it suitable for fail-fast scenarios.
// Note: Item-level validation errors are included in the first error set.
//
// Example:
//
//	schema := Array[string]("tags").Min(1).Max(5)
//	if err := schema.Validate(tags); err != nil {
//		log.Printf("Invalid tags: %v", err)
//	}
func (a *ArraySchema[T]) Validate(value []T) error {
	errs := a.ValidateAll(value)
	if errs.HasErrors() {
		return errs[0]
	}
	return nil
}

// ValidateAll runs all validation rules against the array and returns all errors.
// If validation passes, an empty ValidationErrors slice is returned.
// This method collects all errors including item-level errors, making it suitable
// for scenarios where you want to show all validation problems at once.
// Item-level errors include the array index in the field name (e.g., "emails[1]").
//
// Example:
//
//	emailSchema := String("email").Email()
//	schema := Array[string]("recipients").Items(emailSchema)
//	errs := schema.ValidateAll(recipients)
//	if errs.HasErrors() {
//		for _, err := range errs {
//			fmt.Printf("%s: %s\n", err.Field, err.Message)
//		}
//	}
func (a *ArraySchema[T]) ValidateAll(value []T) ValidationErrors {
	var errs ValidationErrors

	if len(value) == 0 && !a.optional {
		return ValidationErrors{
			NewValidationError(a.field, "is required", "required"),
		}
	}

	if len(value) == 0 && a.optional {
		return errs
	}

	for _, rule := range a.rules {
		if err := rule(value); err != nil {
			errs = append(errs, NewValidationError(a.field, err.Error(), "validation_failed"))
		}
	}

	if a.itemSchema != nil {
		for i, item := range value {
			itemErrs := a.itemSchema.ValidateAll(item)
			for _, err := range itemErrs {
				err.Field = fmt.Sprintf("%s[%d]", a.field, i)
				errs = append(errs, err)
			}
		}
	}

	return errs
}

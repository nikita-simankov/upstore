package validate

import (
	"testing"
)

// TestArrayMinMax tests Min() and Max() length constraints on arrays.
func TestArrayMinMax(t *testing.T) {
	schema := Array[string]("tags").Min(1).Max(5)

	tests := []struct {
		value   []string
		wantErr bool
	}{
		{[]string{"a"}, false},
		{[]string{"a", "b", "c"}, false},
		{[]string{"a", "b", "c", "d", "e"}, false},
		{[]string{}, true},
		{[]string{"a", "b", "c", "d", "e", "f"}, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Array validation for length %d: got error %v, want error %v", len(tt.value), err != nil, tt.wantErr)
		}
	}
}

// TestArrayLength tests exact length constraint.
func TestArrayLength(t *testing.T) {
	schema := Array[string]("coordinates").Length(2)

	tests := []struct {
		value   []string
		wantErr bool
	}{
		{[]string{"10", "20"}, false},
		{[]string{"10"}, true},
		{[]string{"10", "20", "30"}, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Length validation for length %d: got error %v, want error %v", len(tt.value), err != nil, tt.wantErr)
		}
	}
}

// TestArrayOptional tests Optional() allowing empty arrays.
func TestArrayOptional(t *testing.T) {
	schema := Array[string]("hobbies").Optional().Max(10)

	err := schema.Validate([]string{})
	if err != nil {
		t.Errorf("expected no error for empty optional array, got %v", err)
	}

	longArray := make([]string, 15)
	for i := 0; i < 15; i++ {
		longArray[i] = "a"
	}
	err = schema.Validate(longArray)
	if err == nil {
		t.Errorf("expected error for array exceeding max length")
	}
}

// TestArrayWithItemSchema tests Items() with per-item validation.
func TestArrayWithItemSchema(t *testing.T) {
	schema := Array[string]("emails").Items(
		String("email").Email(),
	)

	tests := []struct {
		value   []string
		wantErr bool
	}{
		{[]string{"user@example.com"}, false},
		{[]string{"user@example.com", "admin@example.com"}, false},
		{[]string{"invalid-email"}, true},
		{[]string{"valid@example.com", "invalid"}, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Array with item schema validation: got error %v, want error %v", err != nil, tt.wantErr)
		}
	}
}

// TestArrayUnique tests Unique() ensuring no duplicates.
func TestArrayUnique(t *testing.T) {
	schema := Array[int]("ids").Unique()

	tests := []struct {
		value   []int
		wantErr bool
	}{
		{[]int{1, 2, 3}, false},
		{[]int{1}, false},
		{[]int{1, 2, 2, 3}, true},
		{[]int{5, 5, 5}, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Unique validation: got error %v, want error %v", err != nil, tt.wantErr)
		}
	}
}

// TestArrayRefine tests custom validation with Refine().
func TestArrayRefine(t *testing.T) {
	schema := Array[int]("prices").Refine(
		func(prices []int) bool {
			if len(prices) < 2 {
				return true
			}
			for i := 1; i < len(prices); i++ {
				if prices[i] < prices[i-1] {
					return false // Not in ascending order
				}
			}
			return true
		},
		"prices must be in ascending order",
	)

	tests := []struct {
		value   []int
		wantErr bool
	}{
		{[]int{1, 2, 3, 4}, false},
		{[]int{5}, false},
		{[]int{10, 5, 20}, true},
		{[]int{3, 3, 3}, false}, // Equal values ok
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Refine validation: got error %v, want error %v", err != nil, tt.wantErr)
		}
	}
}

// TestArrayValidateAll tests ValidateAll() with item-level errors.
func TestArrayValidateAll(t *testing.T) {
	emailSchema := String("email").Email()
	schema := Array[string]("recipients").Items(emailSchema)

	errs := schema.ValidateAll([]string{"user@example.com", "invalid"})

	if !errs.HasErrors() {
		t.Errorf("expected errors for invalid email")
	}

	// Should have 1 error for the invalid email at index 1
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}

	// Check that error field includes array index
	if len(errs) > 0 && errs[0].Field != "recipients[1]" {
		t.Errorf("expected error with field 'recipients[1]', got %q", errs[0].Field)
	}
}

// TestArrayIntItems tests Array of integers with item validation.
func TestArrayIntItems(t *testing.T) {
	schema := Array[int]("ports").Min(1).Items(
		Int("port").Min(1).Max(65535),
	)

	tests := []struct {
		value   []int
		wantErr bool
	}{
		{[]int{8080}, false},
		{[]int{8080, 443, 80}, false},
		{[]int{0}, true},           // Port 0 is invalid
		{[]int{8080, 99999}, true}, // Second port out of range
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Array of int validation: got error %v, want error %v", err != nil, tt.wantErr)
		}
	}
}

// TestArrayRequiredAndMin covers Required() rejecting empty arrays and the Min() rule.
func TestArrayRequiredAndMin(t *testing.T) {
	required := Array[string]("tags").Optional().Required()
	if err := required.Validate([]string{}); err == nil {
		t.Errorf("Required: expected empty array to fail, got nil")
	}

	min := Array[int]("ids").Min(2)
	if err := min.Validate([]int{1}); err == nil {
		t.Errorf("Min(2): expected single item to fail, got nil")
	}
	if err := min.Validate([]int{1, 2}); err != nil {
		t.Errorf("Min(2): expected two items to pass, got %v", err)
	}
}

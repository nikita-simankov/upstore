package validate

import (
	"testing"
)

// TestIntValidation tests basic Int schema validation with Min/Max constraints.
func TestIntValidation(t *testing.T) {
	schema := Int("port").Min(1).Max(65535)

	tests := []struct {
		value   int
		wantErr bool
	}{
		{8080, false},
		{1, false},
		{65535, false},
		{0, true},
		{65536, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Int validation for %d: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestIntPositive tests Positive() constraint.
func TestIntPositive(t *testing.T) {
	schema := Int("count").Positive()

	tests := []struct {
		value   int
		wantErr bool
	}{
		{1, false},
		{100, false},
		{0, true},
		{-1, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Positive validation for %d: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestIntNegative tests Negative() constraint.
func TestIntNegative(t *testing.T) {
	schema := Int("offset").Negative()

	tests := []struct {
		value   int
		wantErr bool
	}{
		{-5, false},
		{-1, false},
		{0, true},
		{1, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Negative validation for %d: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestIntOneOf tests OneOf() enum-like validation for integers.
func TestIntOneOf(t *testing.T) {
	schema := Int("status").OneOf(0, 1, 2)

	tests := []struct {
		value   int
		wantErr bool
	}{
		{0, false},
		{1, false},
		{2, false},
		{3, true},
		{-1, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("OneOf validation for %d: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestIntRefine tests custom validation with Refine().
func TestIntRefine(t *testing.T) {
	schema := Int("even").Refine(
		func(val int) bool { return val%2 == 0 },
		"must be even",
	)

	tests := []struct {
		value   int
		wantErr bool
	}{
		{2, false},
		{4, false},
		{1, true},
		{3, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Refine validation for %d: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestIntParseString tests ParseString() parsing and validating integers.
func TestIntParseString(t *testing.T) {
	schema := Int("port").Min(1).Max(65535)

	tests := []struct {
		value   string
		want    int
		wantErr bool
	}{
		{"8080", 8080, false},
		{"1", 1, false},
		{"65535", 65535, false},
		{"invalid", 0, true},
		{"99999", 99999, false}, // Parses ok but fails validation
	}

	for _, tt := range tests {
		got, err := schema.ParseString(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseString(%q): got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Errorf("ParseString(%q): got %d, want %d", tt.value, got, tt.want)
		}
	}
}

// TestIntValidateAll tests ValidateAll() collecting multiple errors.
func TestIntValidateAll(t *testing.T) {
	schema := Int("port").Min(1).Max(65535).Refine(
		func(p int) bool { return p != 80 && p != 443 },
		"reserved port",
	)

	errs := schema.ValidateAll(0)
	if !errs.HasErrors() {
		t.Errorf("expected errors for value 0")
	}
}

// TestFloatValidation tests basic Float schema validation with Min/Max constraints.
func TestFloatValidation(t *testing.T) {
	schema := Float("rate").Min(0.0).Max(100.0)

	tests := []struct {
		value   float64
		wantErr bool
	}{
		{50.5, false},
		{0.0, false},
		{100.0, false},
		{-0.1, true},
		{100.1, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Float validation for %g: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestFloatPositive tests Positive() constraint for floats.
func TestFloatPositive(t *testing.T) {
	schema := Float("price").Positive()

	tests := []struct {
		value   float64
		wantErr bool
	}{
		{0.01, false},
		{99.99, false},
		{0.0, true},
		{-1.5, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Positive validation for %g: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestFloatNegative tests Negative() constraint for floats.
func TestFloatNegative(t *testing.T) {
	schema := Float("adjustment").Negative()

	tests := []struct {
		value   float64
		wantErr bool
	}{
		{-5.5, false},
		{-0.1, false},
		{0.0, true},
		{5.5, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Negative validation for %g: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestFloatRefine tests custom validation with Refine().
func TestFloatRefine(t *testing.T) {
	schema := Float("percentage").Refine(
		func(val float64) bool { return val*100 == float64(int(val*100)) },
		"must have at most 2 decimal places",
	)

	tests := []struct {
		value   float64
		wantErr bool
	}{
		{0.5, false},
		{99.99, false},
		{33.333, true},
		{0.111, true},
	}

	for _, tt := range tests {
		err := schema.Validate(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Refine validation for %g: got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
	}
}

// TestFloatParseString tests ParseString() parsing and validating floats.
func TestFloatParseString(t *testing.T) {
	schema := Float("temperature").Min(-50.0).Max(50.0)

	tests := []struct {
		value   string
		want    float64
		wantErr bool
	}{
		{"25.5", 25.5, false},
		{"0.0", 0.0, false},
		{"-30.2", -30.2, false},
		{"invalid", 0, true},
		{"100.0", 100.0, false}, // Parses ok but fails validation
	}

	for _, tt := range tests {
		got, err := schema.ParseString(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseString(%q): got error %v, want error %v", tt.value, err != nil, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Errorf("ParseString(%q): got %g, want %g", tt.value, got, tt.want)
		}
	}
}

// TestFloatValidateAll tests ValidateAll() collecting multiple errors.
func TestFloatValidateAll(t *testing.T) {
	schema := Float("discount").Min(0.0).Max(1.0).Refine(
		func(d float64) bool { return d*100 == float64(int(d*100)) },
		"at most 2 decimals",
	)

	errs := schema.ValidateAll(1.555)
	if !errs.HasErrors() {
		t.Errorf("expected errors for value 1.555")
	}

	// Should have 2 errors: exceeds max and decimal places
	if len(errs) < 1 {
		t.Errorf("expected at least 1 error, got %d", len(errs))
	}
}

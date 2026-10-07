package validate

import (
	"strings"
)

// Errors aggregates all rule failures from a single Validate call.
// Returns them together, not on the first failure — so the caller sees
// every problem in one pass.
type Errors []error

func (errs Errors) Error() string {
	switch len(errs) {
	case 0:
		return ""
	case 1:
		return errs[0].Error()
	}

	var builder strings.Builder

	for i, e := range errs {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(e.Error())
	}
	return builder.String()
}

func (errs Errors) Unwrap() []error { return errs }

func appendErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	return Errors(errs)
}

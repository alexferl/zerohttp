package validator

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationErrors holds all validation errors for a struct.
// The key is the field path (e.g., "Name", "Address.City", "Items[0].Name").
type ValidationErrors map[string][]string

// Error implements the error interface.
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}

	var parts []string
	for field, errs := range ve {
		if field == "" {
			parts = append(parts, strings.Join(errs, ", "))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", field, strings.Join(errs, ", ")))
		}
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// HasErrors returns true if there are any validation errors.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// FieldErrors returns all errors for a specific field.
func (ve ValidationErrors) FieldErrors(field string) []string {
	return ve[field]
}

// Add adds an error for a specific field.
func (ve ValidationErrors) Add(field, err string) {
	ve[field] = append(ve[field], err)
}

// ValidationErrors returns the errors map (implements ValidationErrorer interface).
func (ve ValidationErrors) ValidationErrors() map[string][]string {
	return ve
}

// ValidationErrorer is implemented by validation error types.
// The default error handler uses this to detect validation errors
// and return 422 Unprocessable Entity with proper formatting.
type ValidationErrorer interface {
	error
	ValidationErrors() map[string][]string
}

// Ensure ValidationErrors implements ValidationErrorer
var _ ValidationErrorer = (ValidationErrors)(nil)

// BindError wraps binding errors to distinguish them from validation errors.
// The default error handler uses this to return 400 instead of 422.
type BindError struct {
	Err error
}

func (e *BindError) Error() string {
	return "bind error: " + e.Err.Error()
}

func (e *BindError) Unwrap() error {
	return e.Err
}

// IsBindError checks if an error is a binding error.
func IsBindError(err error) bool {
	if err == nil {
		return false
	}
	var be *BindError
	return errors.As(err, &be)
}

// Package errors provides shared error types to avoid import cycles
// between the main zerohttp package and middleware.
package errors

import "errors"

// ValidationErrorer is the interface for validation errors.
type ValidationErrorer interface {
	error
	ValidationErrors() map[string][]string
}

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

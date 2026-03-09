package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestBindError(t *testing.T) {
	inner := errors.New("invalid JSON")
	be := &BindError{Err: inner}

	// Test Error() message
	if msg := be.Error(); msg != "bind error: invalid JSON" {
		t.Errorf("expected 'bind error: invalid JSON', got %q", msg)
	}

	// Test Unwrap
	if unwrapped := be.Unwrap(); unwrapped != inner {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestIsBindError(t *testing.T) {
	// Test with nil
	if IsBindError(nil) {
		t.Error("expected IsBindError(nil) to be false")
	}

	// Test with regular error
	regularErr := errors.New("some error")
	if IsBindError(regularErr) {
		t.Error("expected IsBindError(regularErr) to be false")
	}

	// Test with BindError
	bindErr := &BindError{Err: errors.New("bind error")}
	if !IsBindError(bindErr) {
		t.Error("expected IsBindError(bindErr) to be true")
	}

	// Test with wrapped BindError
	wrappedErr := fmt.Errorf("wrapped: %w", bindErr)
	if !IsBindError(wrappedErr) {
		t.Error("expected IsBindError(wrappedErr) to be true")
	}
}

func TestValidationErrorer(t *testing.T) {
	// Test that ValidationErrorer interface works
	var ve ValidationErrorer = &testValidationError{
		errs: map[string][]string{"field": {"required"}},
	}

	if errs := ve.ValidationErrors(); len(errs) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(errs))
	}
}

type testValidationError struct {
	errs map[string][]string
}

func (t *testValidationError) Error() string {
	return "validation failed"
}

func (t *testValidationError) ValidationErrors() map[string][]string {
	return t.errs
}

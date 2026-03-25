package validator

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type TestUser struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=13,max=120"`
}

func TestValidationErrors_ValidUser(t *testing.T) {
	input := TestUser{Name: "John", Email: "john@example.com", Age: 25}
	err := New().Struct(&input)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidationErrors_MissingRequired(t *testing.T) {
	input := TestUser{}
	err := New().Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		t.Errorf("expected ValidationErrors, got %T", err)
		return
	}
	if len(ve.FieldErrors("Name")) == 0 {
		t.Errorf("expected Name error, got none")
	}
	if len(ve.FieldErrors("Email")) == 0 {
		t.Errorf("expected Email error, got none")
	}
}

func TestValidationErrors_InvalidEmail(t *testing.T) {
	input := TestUser{Name: "John", Email: "not-an-email", Age: 25}
	err := New().Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	errs := ve.FieldErrors("Email")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "invalid email") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected invalid email error, got %v", errs)
	}
}

func TestValidationErrors_MinLength(t *testing.T) {
	input := TestUser{Name: "J", Email: "john@example.com", Age: 25}
	err := New().Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	errs := ve.FieldErrors("Name")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "at least 2") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected min length error, got %v", errs)
	}
}

func TestValidationErrors_MaxLength(t *testing.T) {
	input := TestUser{Name: strings.Repeat("a", 51), Email: "john@example.com", Age: 25}
	err := New().Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	errs := ve.FieldErrors("Name")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "at most 50") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected max length error, got %v", errs)
	}
}

func TestValidationErrors_Multiple(t *testing.T) {
	input := TestUser{Name: "", Email: "bad", Age: 5}
	err := New().Struct(&input)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	var ve ValidationErrors
	errors.As(err, &ve)
	if len(ve.FieldErrors("Name")) == 0 {
		t.Errorf("expected Name error")
	}
	if len(ve.FieldErrors("Email")) == 0 {
		t.Errorf("expected Email error")
	}
	if len(ve.FieldErrors("Age")) == 0 {
		t.Errorf("expected Age error")
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name         string
		errors       ValidationErrors
		wantExact    string
		wantContains []string
	}{
		{
			name:      "empty errors",
			errors:    ValidationErrors{},
			wantExact: "validation failed",
		},
		{
			name:      "single field error",
			errors:    ValidationErrors{"name": {"required"}},
			wantExact: "validation failed: name: required",
		},
		{
			name:         "multiple field errors",
			errors:       ValidationErrors{"name": {"required"}, "email": {"invalid format"}},
			wantContains: []string{"name: required", "email: invalid format"},
		},
		{
			name:      "multiple errors per field",
			errors:    ValidationErrors{"password": {"too short", "must contain number"}},
			wantExact: "validation failed: password: too short, must contain number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.Error()
			if tt.wantExact != "" && got != tt.wantExact {
				t.Errorf("Error() = %q, want %q", got, tt.wantExact)
			}
			for _, substr := range tt.wantContains {
				if !strings.Contains(got, substr) {
					t.Errorf("Error() = %q, should contain %q", got, substr)
				}
			}
		})
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	ve := ValidationErrors{}
	if ve.HasErrors() {
		t.Error("expected HasErrors to be false for empty errors")
	}
	ve.Add("field", "error")
	if !ve.HasErrors() {
		t.Error("expected HasErrors to be true when errors exist")
	}
}

func TestValidationErrors_ValidationErrors(t *testing.T) {
	ve := ValidationErrors{
		"Name": {"required"},
		"Age":  {"min"},
	}
	errs := ve.ValidationErrors()
	if len(errs["Name"]) != 1 || errs["Name"][0] != "required" {
		t.Errorf("expected Name error, got %v", errs["Name"])
	}
	if len(errs["Age"]) != 1 || errs["Age"][0] != "min" {
		t.Errorf("expected Age error, got %v", errs["Age"])
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

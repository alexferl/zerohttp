package validator

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

type TestUser struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=13,max=120"`
}

func TestValidationErrors_ValidUser(t *testing.T) {
	input := TestUser{Name: "John", Email: "john@example.com", Age: 25}
	err := New().Struct(&input)
	zhtest.AssertNoError(t, err)
}

func TestValidationErrors_MissingRequired(t *testing.T) {
	input := TestUser{}
	err := New().Struct(&input)
	zhtest.AssertError(t, err)
	var ve ValidationErrors
	zhtest.AssertTrue(t, errors.As(err, &ve))
	zhtest.AssertNotEqual(t, 0, len(ve.FieldErrors("Name")))
	zhtest.AssertNotEqual(t, 0, len(ve.FieldErrors("Email")))
}

func TestValidationErrors_InvalidEmail(t *testing.T) {
	input := TestUser{Name: "John", Email: "not-an-email", Age: 25}
	err := New().Struct(&input)
	zhtest.AssertError(t, err)
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
	zhtest.AssertTrue(t, found)
}

func TestValidationErrors_MinLength(t *testing.T) {
	input := TestUser{Name: "J", Email: "john@example.com", Age: 25}
	err := New().Struct(&input)
	zhtest.AssertError(t, err)
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
	zhtest.AssertTrue(t, found)
}

func TestValidationErrors_MaxLength(t *testing.T) {
	input := TestUser{Name: strings.Repeat("a", 51), Email: "john@example.com", Age: 25}
	err := New().Struct(&input)
	zhtest.AssertError(t, err)
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
	zhtest.AssertTrue(t, found)
}

func TestValidationErrors_Multiple(t *testing.T) {
	input := TestUser{Name: "", Email: "bad", Age: 5}
	err := New().Struct(&input)
	zhtest.AssertError(t, err)
	var ve ValidationErrors
	errors.As(err, &ve)
	zhtest.AssertNotEqual(t, 0, len(ve.FieldErrors("Name")))
	zhtest.AssertNotEqual(t, 0, len(ve.FieldErrors("Email")))
	zhtest.AssertNotEqual(t, 0, len(ve.FieldErrors("Age")))
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
			if tt.wantExact != "" {
				zhtest.AssertEqual(t, tt.wantExact, got)
			}
			for _, substr := range tt.wantContains {
				zhtest.AssertTrue(t, strings.Contains(got, substr))
			}
		})
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	ve := ValidationErrors{}
	zhtest.AssertFalse(t, ve.HasErrors())
	ve.Add("field", "error")
	zhtest.AssertTrue(t, ve.HasErrors())
}

func TestValidationErrors_ValidationErrors(t *testing.T) {
	ve := ValidationErrors{
		"Name": {"required"},
		"Age":  {"min"},
	}
	errs := ve.ValidationErrors()
	zhtest.AssertEqual(t, 1, len(errs["Name"]))
	zhtest.AssertEqual(t, "required", errs["Name"][0])
	zhtest.AssertEqual(t, 1, len(errs["Age"]))
	zhtest.AssertEqual(t, "min", errs["Age"][0])
}

func TestValidationErrorer(t *testing.T) {
	// Test that ValidationErrorer interface works
	var ve ValidationErrorer = &testValidationError{
		errs: map[string][]string{"field": {"required"}},
	}

	zhtest.AssertEqual(t, 1, len(ve.ValidationErrors()))
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
	zhtest.AssertEqual(t, "bind error: invalid JSON", be.Error())

	// Test Unwrap
	zhtest.AssertEqual(t, inner, be.Unwrap())
}

func TestIsBindError(t *testing.T) {
	// Test with nil
	zhtest.AssertFalse(t, IsBindError(nil))

	// Test with regular error
	regularErr := errors.New("some error")
	zhtest.AssertFalse(t, IsBindError(regularErr))

	// Test with BindError
	bindErr := &BindError{Err: errors.New("bind error")}
	zhtest.AssertTrue(t, IsBindError(bindErr))

	// Test with wrapped BindError
	wrappedErr := fmt.Errorf("wrapped: %w", bindErr)
	zhtest.AssertTrue(t, IsBindError(wrappedErr))
}

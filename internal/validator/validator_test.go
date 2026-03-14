package validator

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		// Basic types
		{"string empty", "", true},
		{"string non-empty", "hello", false},
		{"int zero", 0, true},
		{"int non-zero", 42, false},
		{"uint zero", uint(0), true},
		{"uint non-zero", uint(42), false},
		{"float zero", 0.0, true},
		{"float non-zero", 3.14, false},
		{"bool false", false, true},
		{"bool true", true, false},
		{"slice empty", []int{}, true},
		{"slice non-empty", []int{1, 2, 3}, false},
		{"map empty", map[string]int{}, true},
		{"map non-empty", map[string]int{"a": 1}, false},
		{"ptr nil", (*string)(nil), true},
		{"ptr non-nil", func() *int { i := 5; return &i }(), false},
		{"struct zero", struct{ Name string }{}, true},
		{"struct non-zero", struct{ Name string }{Name: "John"}, false},
		// Edge cases that hit default case
		{"channel non-nil", make(chan int), false},
		{"function non-nil", func() {}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.input)
			result := isZeroValue(v)
			if result != tt.expected {
				t.Errorf("isZeroValue(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}

	// Test interface types (need special handling)
	t.Run("interface non-nil", func(t *testing.T) {
		var iface any = "hello"
		v := reflect.ValueOf(&iface).Elem()
		if isZeroValue(v) {
			t.Error("expected non-empty interface to not be zero")
		}
	})

	t.Run("interface nil", func(t *testing.T) {
		var nilIface any
		v := reflect.ValueOf(&nilIface).Elem()
		if !isZeroValue(v) {
			t.Error("expected nil interface to be zero")
		}
	})
}

// TestGetJSONFieldName tests the JSON field name extraction from struct tags
func TestGetJSONFieldName(t *testing.T) {
	tests := []struct {
		name      string
		jsonTag   string
		fieldName string
		expected  string
	}{
		{
			name:      "simple json tag",
			jsonTag:   "username",
			fieldName: "Username",
			expected:  "username",
		},
		{
			name:      "json tag with omitempty",
			jsonTag:   "username,omitempty",
			fieldName: "Username",
			expected:  "username",
		},
		{
			name:      "json tag with multiple options",
			jsonTag:   "created_at,omitempty,string",
			fieldName: "CreatedAt",
			expected:  "created_at",
		},
		{
			name:      "empty json tag",
			jsonTag:   "",
			fieldName: "Name",
			expected:  "Name",
		},
		{
			name:      "json tag with dash (ignored field)",
			jsonTag:   "-",
			fieldName: "Internal",
			expected:  "Internal",
		},
		{
			name:      "json tag with only options (empty name)",
			jsonTag:   ",omitempty",
			fieldName: "Description",
			expected:  "Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a struct field with the desired tag
			field := reflect.StructField{
				Name: tt.fieldName,
				Tag:  reflect.StructTag(`json:"` + tt.jsonTag + `"`),
			}

			result := getJSONFieldName(field)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseTagEdgeCases(t *testing.T) {
	// Test with empty parts in tag (comma without validator)
	type TestTag struct {
		Field string `validate:"required,,min=3"`
	}

	input := TestTag{Field: "ab"}
	err := NewValidator().Struct(&input)
	// Should still validate required and min, skipping empty part
	if err == nil {
		t.Error("expected error for min validation")
	}
}

func TestStruct_NilCases(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		err := NewValidator().Struct(nil)
		if err == nil {
			t.Error("expected error for nil value")
		}
		var ve ValidationErrors
		ok := errors.As(err, &ve)
		if !ok {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		if errs := ve.FieldErrors(""); len(errs) == 0 || errs[0] != "nil value" {
			t.Errorf("expected nil value error, got %v", errs)
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		var ptr *struct{ Name string }
		err := NewValidator().Struct(ptr)
		if err == nil {
			t.Error("expected error for nil pointer")
		}
		var ve ValidationErrors
		ok := errors.As(err, &ve)
		if !ok {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		if errs := ve.FieldErrors(""); len(errs) == 0 || errs[0] != "nil pointer" {
			t.Errorf("expected nil pointer error, got %v", errs)
		}
	})

	t.Run("non-struct value", func(t *testing.T) {
		err := NewValidator().Struct("not a struct")
		if err == nil {
			t.Error("expected error for non-struct value")
		}
		var ve ValidationErrors
		ok := errors.As(err, &ve)
		if !ok {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		if errs := ve.FieldErrors(""); len(errs) == 0 || !strings.Contains(errs[0], "expected struct") {
			t.Errorf("expected struct error, got %v", errs)
		}
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		s := "hello"
		err := NewValidator().Struct(&s)
		if err == nil {
			t.Error("expected error for pointer to non-struct")
		}
		var ve ValidationErrors
		ok := errors.As(err, &ve)
		if !ok {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		if errs := ve.FieldErrors(""); len(errs) == 0 || !strings.Contains(errs[0], "expected struct") {
			t.Errorf("expected struct error, got %v", errs)
		}
	})
}

// TestValidatorInterface tests struct implementing Validate() error
type ValidatableUser struct {
	Name  string
	Email string
}

func (u *ValidatableUser) Validate() error {
	if u.Name == "" {
		return fmt.Errorf("name is required")
	}
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("invalid email")
	}
	return nil
}

func TestValidatorInterface(t *testing.T) {
	type TestValidatable struct {
		User ValidatableUser
	}

	t.Run("valid user", func(t *testing.T) {
		input := TestValidatable{User: ValidatableUser{Name: "John", Email: "john@example.com"}}
		err := NewValidator().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid user - missing name", func(t *testing.T) {
		input := TestValidatable{User: ValidatableUser{Email: "john@example.com"}}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if errs := ve.FieldErrors("User"); len(errs) == 0 {
			t.Errorf("expected User validation error, got %v", ve)
		}
	})

	t.Run("invalid user - bad email", func(t *testing.T) {
		input := TestValidatable{User: ValidatableUser{Name: "John", Email: "not-an-email"}}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if errs := ve.FieldErrors("User"); len(errs) == 0 {
			t.Errorf("expected User validation error, got %v", ve)
		}
	})
}

// TestUnknownValidatorEdgeCases tests unknown validator handling
func TestUnknownValidatorEdgeCases(t *testing.T) {
	t.Run("unknown validator on field", func(t *testing.T) {
		type TestUnknown struct {
			Value string `validate:"unknown_validator"`
		}
		input := TestUnknown{Value: "test"}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error for unknown validator")
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		errs := ve.FieldErrors("Value")
		if len(errs) == 0 {
			t.Errorf("expected error for Value field, got: %v", ve)
		}
	})

	t.Run("unknown validator in each slice", func(t *testing.T) {
		type TestUnknownEach struct {
			Items []string `validate:"each,nonexistent"`
		}
		input := TestUnknownEach{Items: []string{"a", "b"}}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error for unknown validator in each")
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		errs := ve.FieldErrors("Items[0]")
		if len(errs) == 0 {
			t.Errorf("expected error for Items[0], got: %v", ve)
		}
	})

	t.Run("unknown validator in each map", func(t *testing.T) {
		type TestUnknownEachMap struct {
			Items map[string]string `validate:"each,nonexistent"`
		}
		input := TestUnknownEachMap{Items: map[string]string{"key": "value"}}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error for unknown validator in each map")
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		errs := ve.FieldErrors("Items[key]")
		if len(errs) == 0 {
			t.Errorf("expected error for Items[key], got: %v", ve)
		}
	})
}

// TestEmptyStructValidation tests empty struct handling
func TestEmptyStructValidation(t *testing.T) {
	t.Run("empty struct with no tags", func(t *testing.T) {
		type EmptyStruct struct{}
		input := EmptyStruct{}
		err := NewValidator().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for empty struct: %v", err)
		}
	})

	t.Run("struct with only untagged fields", func(t *testing.T) {
		type UntaggedStruct struct {
			Name  string
			Value int
		}
		input := UntaggedStruct{Name: "", Value: 0}
		err := NewValidator().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for untagged struct: %v", err)
		}
	})

	t.Run("struct with only skipped fields", func(t *testing.T) {
		type SkippedStruct struct {
			Name  string `validate:"-"`
			Value int    `validate:"-"`
		}
		input := SkippedStruct{Name: "", Value: 0}
		err := NewValidator().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for skipped struct: %v", err)
		}
	})
}

// TestDeepNesting tests deeply nested struct validation
func TestDeepNesting(t *testing.T) {
	t.Run("three level nesting", func(t *testing.T) {
		type Level3 struct {
			Name string `validate:"required"`
		}
		type Level2 struct {
			Level3 Level3 `validate:"each"`
		}
		type Level1 struct {
			Level2 Level2 `validate:"each"`
		}
		type Root struct {
			Level1 Level1 `validate:"each"`
		}

		input := Root{
			Level1: Level1{
				Level2: Level2{
					Level3: Level3{Name: ""},
				},
			},
		}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error for deeply nested validation failure")
		}
		var ve ValidationErrors
		if errors.As(err, &ve) {
			errs := ve.FieldErrors("Level1.Level2.Level3.Name")
			if len(errs) == 0 {
				t.Errorf("expected error for deep path, got: %v", ve)
			}
		}
	})

	t.Run("nested slice of structs", func(t *testing.T) {
		type Item struct {
			Name string `validate:"required"`
		}
		type Container struct {
			Items []Item `validate:"each"`
		}
		type Root struct {
			Containers []Container `validate:"each"`
		}

		input := Root{
			Containers: []Container{
				{Items: []Item{{Name: "valid"}}},
				{Items: []Item{{Name: ""}}},
			},
		}
		err := NewValidator().Struct(&input)
		if err == nil {
			t.Error("expected error for nested slice validation failure")
		}
	})
}

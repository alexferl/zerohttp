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
	err := New().Struct(&input)
	// Should still validate required and min, skipping empty part
	if err == nil {
		t.Error("expected error for min validation")
	}
}

func TestStruct_NilCases(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		err := New().Struct(nil)
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
		err := New().Struct(ptr)
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
		err := New().Struct("not a struct")
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
		err := New().Struct(&s)
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
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid user - missing name", func(t *testing.T) {
		input := TestValidatable{User: ValidatableUser{Email: "john@example.com"}}
		err := New().Struct(&input)
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
		err := New().Struct(&input)
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

func TestUnknownValidatorEdgeCases(t *testing.T) {
	t.Run("unknown validator on field", func(t *testing.T) {
		type TestUnknown struct {
			Value string `validate:"unknown_validator"`
		}
		input := TestUnknown{Value: "test"}
		err := New().Struct(&input)
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
		err := New().Struct(&input)
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
		err := New().Struct(&input)
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

func TestEmptyStructValidation(t *testing.T) {
	t.Run("empty struct with no tags", func(t *testing.T) {
		type EmptyStruct struct{}
		input := EmptyStruct{}
		err := New().Struct(&input)
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
		err := New().Struct(&input)
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
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for skipped struct: %v", err)
		}
	})
}

type testUser struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=13,max=120"`
}

type testOptional struct {
	Name  string `validate:"omitempty,min=2"`
	Email string `validate:"omitempty,email"`
}

type testPointers struct {
	Name *string `validate:"required,min=2"`
	Age  *int    `validate:"omitempty,min=13"`
}

type testNested struct {
	User    testUser `validate:"required"`
	Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
	}
}

type testSlice struct {
	Tags  []string `validate:"min=1,max=5"`
	Items []struct {
		Name  string `validate:"required"`
		Price int    `validate:"min=0"`
	}
}

// Custom validator test
type testCustom struct {
	Code string `validate:"custom_code"`
}

func TestOmitempty(t *testing.T) {
	tests := []struct {
		name    string
		input   testOptional
		wantErr bool
	}{
		{
			name:    "empty is valid",
			input:   testOptional{},
			wantErr: false,
		},
		{
			name:    "valid name",
			input:   testOptional{Name: "John"},
			wantErr: false,
		},
		{
			name:    "valid email",
			input:   testOptional{Email: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "short name fails",
			input:   testOptional{Name: "J"},
			wantErr: true,
		},
		{
			name:    "bad email fails",
			input:   testOptional{Email: "not-an-email"},
			wantErr: true,
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestPointerFields(t *testing.T) {
	name := "John"
	shortName := "J"
	age := 25

	tests := []struct {
		name    string
		input   testPointers
		wantErr bool
		errors  map[string][]string
	}{
		{
			name:    "nil required pointer fails",
			input:   testPointers{},
			wantErr: true,
			errors: map[string][]string{
				"Name": {"required"},
			},
		},
		{
			name:    "valid required pointer",
			input:   testPointers{Name: &name},
			wantErr: false,
		},
		{
			name:    "short required pointer fails",
			input:   testPointers{Name: &shortName},
			wantErr: true,
			errors: map[string][]string{
				"Name": {"must be at least 2 characters"},
			},
		},
		{
			name:    "nil optional pointer is ok",
			input:   testPointers{Name: &name, Age: nil},
			wantErr: false,
		},
		{
			name:    "valid optional pointer",
			input:   testPointers{Name: &name, Age: &age},
			wantErr: false,
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr {
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
				for field, expectedErrs := range tt.errors {
					actualErrs := ve.FieldErrors(field)
					for _, expected := range expectedErrs {
						found := false
						for _, actual := range actualErrs {
							if strings.Contains(actual, expected) {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected error containing %q for field %s, got %v", expected, field, actualErrs)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestNestedStructs(t *testing.T) {
	tests := []struct {
		name    string
		input   testNested
		wantErr bool
		errors  map[string][]string
	}{
		{
			name: "valid nested",
			input: testNested{
				User: testUser{Name: "John", Email: "john@example.com", Age: 25},
				Address: struct {
					Street string `validate:"required"`
					City   string `validate:"required"`
				}{Street: "123 Main St", City: "NYC"},
			},
			wantErr: false,
		},
		{
			name:    "missing nested fields",
			input:   testNested{},
			wantErr: true,
			errors: map[string][]string{
				"User.Name":      {"required"},
				"User.Email":     {"required"},
				"Address.Street": {"required"},
				"Address.City":   {"required"},
			},
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr {
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
				for field, expectedErrs := range tt.errors {
					actualErrs := ve.FieldErrors(field)
					if len(actualErrs) == 0 {
						t.Errorf("expected errors for field %s, got none. All errors: %v", field, ve)
						continue
					}
					for _, expected := range expectedErrs {
						found := false
						for _, actual := range actualErrs {
							if strings.Contains(actual, expected) {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected error containing %q for field %s, got %v", expected, field, actualErrs)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSliceValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   testSlice
		wantErr bool
		errors  map[string][]string
	}{
		{
			name:    "empty slice fails min",
			input:   testSlice{},
			wantErr: true,
			errors: map[string][]string{
				"Tags": {"must have at least 1 items"},
			},
		},
		{
			name: "valid slice",
			input: testSlice{
				Tags: []string{"a", "b"},
			},
			wantErr: false,
		},
		{
			name: "too many tags",
			input: testSlice{
				Tags: []string{"a", "b", "c", "d", "e", "f"},
			},
			wantErr: true,
			errors: map[string][]string{
				"Tags": {"must have at most 5 items"},
			},
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr {
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
				for field, expectedErrs := range tt.errors {
					actualErrs := ve.FieldErrors(field)
					if len(actualErrs) == 0 {
						t.Errorf("expected errors for field %s, got none. All errors: %v", field, ve)
						continue
					}
					for _, expected := range expectedErrs {
						found := false
						for _, actual := range actualErrs {
							if strings.Contains(actual, expected) {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected error containing %q for field %s, got %v", expected, field, actualErrs)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestCustomValidator(t *testing.T) {
	// Register custom validator
	v := New()
	v.Register("custom_code", func(value reflect.Value, param string) error {
		if value.Kind() != reflect.String {
			return errors.New("custom_code only supports strings")
		}
		code := value.String()
		if len(code) != 5 {
			return errors.New("code must be 5 characters")
		}
		return nil
	})

	tests := []struct {
		name    string
		input   testCustom
		wantErr bool
	}{
		{
			name:    "valid code",
			input:   testCustom{Code: "ABC12"},
			wantErr: false,
		},
		{
			name:    "invalid code length",
			input:   testCustom{Code: "ABC1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// rootValidatableOrder is used to test root struct validation
type rootValidatableOrder struct {
	Items    []string `validate:"required,min=1"`
	Total    float64  `validate:"gte=0"`
	Discount float64  `validate:"gte=0"`
}

// Validate implements custom cross-field validation on the root struct
func (o rootValidatableOrder) Validate() error {
	var sum float64
	for range o.Items {
		sum += 10.0 // simplified pricing
	}
	if o.Total != sum {
		return fmt.Errorf("total must equal sum of items")
	}
	if o.Discount > o.Total {
		return fmt.Errorf("discount cannot exceed total")
	}
	return nil
}

// TestRootStructValidate tests that Validate() is called on the root struct itself
func TestRootStructValidate(t *testing.T) {
	v := New()

	t.Run("valid order", func(t *testing.T) {
		input := rootValidatableOrder{
			Items:    []string{"item1", "item2"},
			Total:    20.0,
			Discount: 5.0,
		}
		err := v.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid total - cross field validation fails", func(t *testing.T) {
		input := rootValidatableOrder{
			Items: []string{"item1", "item2"},
			Total: 100.0, // Wrong total
		}
		err := v.Struct(&input)
		if err == nil {
			t.Error("expected error for mismatched total")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		// Root-level validation errors use the struct type name
		if errs := ve.FieldErrors("rootValidatableOrder"); len(errs) == 0 {
			t.Errorf("expected root validation error, got %v", ve)
		}
	})

	t.Run("discount exceeds total", func(t *testing.T) {
		input := rootValidatableOrder{
			Items:    []string{"item1"},
			Total:    10.0,
			Discount: 20.0, // More than total
		}
		err := v.Struct(&input)
		if err == nil {
			t.Error("expected error for discount exceeding total")
		}
	})
}

func TestEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		Name string `validate:"required"`
	}
	type testEmbedded struct {
		Embedded
		Age int `validate:"min=0"`
	}

	v := New()

	t.Run("valid embedded", func(t *testing.T) {
		input := testEmbedded{Embedded: Embedded{Name: "John"}, Age: 25}
		err := v.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid embedded field", func(t *testing.T) {
		input := testEmbedded{Age: 25}
		err := v.Struct(&input)
		if err == nil {
			t.Error("expected error for missing Name in embedded struct")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if errs := ve.FieldErrors("Name"); len(errs) == 0 {
			t.Errorf("expected Name error, got %v", ve)
		}
	})
}

func TestNestedPointerValidation(t *testing.T) {
	type Inner struct {
		Name string `validate:"required"`
	}
	type testNestedPtr struct {
		Inner *Inner `validate:"required"`
	}

	v := New()

	t.Run("nil required pointer", func(t *testing.T) {
		input := testNestedPtr{Inner: nil}
		err := v.Struct(&input)
		if err == nil {
			t.Error("expected error for nil required pointer")
		}
	})

	t.Run("valid pointer", func(t *testing.T) {
		input := testNestedPtr{Inner: &Inner{Name: "John"}}
		err := v.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid inner struct", func(t *testing.T) {
		input := testNestedPtr{Inner: &Inner{Name: ""}}
		err := v.Struct(&input)
		if err == nil {
			t.Error("expected error for invalid inner struct")
		}
	})
}

func TestPointerWithOmitEmpty(t *testing.T) {
	type testPtrOmit struct {
		Name  *string `validate:"omitempty,min=2"`
		Email *string `validate:"omitempty,email"`
	}

	v := New()

	t.Run("nil pointer with omitempty is valid", func(t *testing.T) {
		input := testPtrOmit{}
		err := v.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid pointer with omitempty", func(t *testing.T) {
		name := "John"
		email := "john@example.com"
		input := testPtrOmit{Name: &name, Email: &email}
		err := v.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid short name with omitempty", func(t *testing.T) {
		name := "J"
		input := testPtrOmit{Name: &name}
		err := v.Struct(&input)
		if err == nil {
			t.Error("expected error for short name")
		}
	})
}

func TestJSONFieldNameInErrors(t *testing.T) {
	type testRequest struct {
		UserName string `json:"user_name" validate:"required,min=5"`
		Email    string `json:"email_address" validate:"required,email"`
	}

	v := New()
	input := testRequest{
		UserName: "ab",  // too short
		Email:    "bad", // invalid email
	}

	err := v.Struct(&input)
	if err == nil {
		t.Fatal("expected error")
	}

	var ve ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// Errors should use json tag names, not Go field names
	if errs := ve.FieldErrors("user_name"); len(errs) == 0 {
		t.Errorf("expected error on 'user_name' (json tag), got errors: %v", ve)
	}
	if errs := ve.FieldErrors("Username"); len(errs) > 0 {
		t.Errorf("should not have error on 'Username' (Go field name), got: %v", errs)
	}

	if errs := ve.FieldErrors("email_address"); len(errs) == 0 {
		t.Errorf("expected error on 'email_address' (json tag), got errors: %v", ve)
	}
	if errs := ve.FieldErrors("Email"); len(errs) > 0 {
		t.Errorf("should not have error on 'Email' (Go field name), got: %v", errs)
	}
}

// TestJSONFieldNameInNestedErrors verifies json tag names in nested structs
func TestJSONFieldNameInNestedErrors(t *testing.T) {
	type Address struct {
		Street string `json:"street_address" validate:"required"`
		City   string `json:"city_name" validate:"required"`
	}
	type Person struct {
		Name    string  `json:"full_name" validate:"required"`
		Address Address `json:"home_address"`
	}

	v := New()
	input := Person{
		Name:    "",
		Address: Address{Street: "", City: "NYC"},
	}

	err := v.Struct(&input)
	if err == nil {
		t.Fatal("expected error")
	}

	var ve ValidationErrors
	ok := errors.As(err, &ve)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// Check nested paths use json tag names
	if errs := ve.FieldErrors("full_name"); len(errs) == 0 {
		t.Errorf("expected error on 'full_name', got errors: %v", ve)
	}
	if errs := ve.FieldErrors("home_address.street_address"); len(errs) == 0 {
		t.Errorf("expected error on 'home_address.street_address', got errors: %v", ve)
	}
}

// TestAnonymousEmbeddedStruct tests validation of anonymous embedded structs
func TestAnonymousEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		Value string `validate:"required"`
	}

	type testAnonymous struct {
		Embedded
		Name string `validate:"required"`
	}

	tests := []struct {
		name     string
		input    testAnonymous
		wantErr  bool
		errField string
	}{
		{
			name:    "all valid",
			input:   testAnonymous{Embedded: Embedded{Value: "embedded"}, Name: "test"},
			wantErr: false,
		},
		{
			name:     "embedded field invalid",
			input:    testAnonymous{Embedded: Embedded{Value: ""}, Name: "test"},
			wantErr:  true,
			errField: "Value",
		},
		{
			name:     "regular field invalid",
			input:    testAnonymous{Embedded: Embedded{Value: "embedded"}, Name: ""},
			wantErr:  true,
			errField: "Name",
		},
		{
			name:     "both invalid",
			input:    testAnonymous{Embedded: Embedded{Value: ""}, Name: ""},
			wantErr:  true,
			errField: "Value",
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr {
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
				errs := ve.FieldErrors(tt.errField)
				if len(errs) == 0 {
					t.Errorf("expected error for field %s, got none", tt.errField)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestAnonymousEmbeddedStructWithJSONTags tests anonymous embedded structs with json tags
func TestAnonymousEmbeddedStructWithJSONTags(t *testing.T) {
	type Embedded struct {
		Value string `json:"embedded_value" validate:"required"`
	}

	type testAnonymous struct {
		Embedded
		Name string `json:"name" validate:"required"`
	}

	tests := []struct {
		name     string
		input    testAnonymous
		wantErr  bool
		errField string
	}{
		{
			name:    "all valid",
			input:   testAnonymous{Embedded: Embedded{Value: "embedded"}, Name: "test"},
			wantErr: false,
		},
		{
			name:     "embedded field uses json tag",
			input:    testAnonymous{Embedded: Embedded{Value: ""}, Name: "test"},
			wantErr:  true,
			errField: "embedded_value",
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.input)
			if tt.wantErr {
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
				errs := ve.FieldErrors(tt.errField)
				if len(errs) == 0 {
					t.Errorf("expected error for field %s, got none", tt.errField)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestValidationErrorsMethods tests ValidationErrors helper methods
func TestValidationErrorsMethods(t *testing.T) {
	t.Run("HasErrors with errors", func(t *testing.T) {
		type Test struct {
			Value string `validate:"required"`
		}
		v := New()
		input := Test{Value: ""}
		err := v.Struct(&input)
		if err == nil {
			t.Fatal("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if !ve.HasErrors() {
			t.Error("HasErrors should return true when there are errors")
		}
	})

	t.Run("FieldErrors for non-existent field", func(t *testing.T) {
		type Test struct {
			Value string `validate:"required"`
		}
		v := New()
		input := Test{Value: ""}
		err := v.Struct(&input)
		if err == nil {
			t.Fatal("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		errs := ve.FieldErrors("nonexistent")
		if len(errs) != 0 {
			t.Errorf("expected empty slice for non-existent field, got: %v", errs)
		}
	})

	t.Run("Add error manually", func(t *testing.T) {
		ve := make(ValidationErrors)
		ve.Add("field1", "error 1")
		ve.Add("field1", "error 2")
		ve.Add("field2", "error 3")

		if len(ve["field1"]) != 2 {
			t.Errorf("expected 2 errors for field1, got: %d", len(ve["field1"]))
		}
		if len(ve["field2"]) != 1 {
			t.Errorf("expected 1 error for field2, got: %d", len(ve["field2"]))
		}
	})

	t.Run("Error string format", func(t *testing.T) {
		ve := make(ValidationErrors)
		ve.Add("field1", "error 1")
		ve.Add("field2", "error 2")

		msg := ve.Error()
		if msg == "" {
			t.Error("Error() should return non-empty string")
		}
		if msg == "validation failed" {
			t.Error("Error() should include field details when there are errors")
		}
	})

	t.Run("Error on empty ValidationErrors", func(t *testing.T) {
		ve := ValidationErrors{}
		msg := ve.Error()
		if msg != "validation failed" {
			t.Errorf("expected 'validation failed', got: %s", msg)
		}
	})

	t.Run("ValidationErrors map accessor", func(t *testing.T) {
		ve := make(ValidationErrors)
		ve.Add("field", "error")

		m := ve.ValidationErrors()
		if len(m["field"]) != 1 {
			t.Errorf("expected 1 error, got: %d", len(m["field"]))
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
		err := New().Struct(&input)
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
		err := New().Struct(&input)
		if err == nil {
			t.Error("expected error for nested slice validation failure")
		}
	})
}

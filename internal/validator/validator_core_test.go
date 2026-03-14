package validator

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRequiredValidator(t *testing.T) {
	type TestRequired struct {
		Name  string `validate:"required"`
		Email string `validate:"required"`
		Age   int    `validate:"required"`
	}

	tests := []struct {
		name    string
		input   TestRequired
		wantErr bool
	}{
		{
			name:    "all fields present",
			input:   TestRequired{Name: "John", Email: "john@example.com", Age: 25},
			wantErr: false,
		},
		{
			name:    "empty string fails",
			input:   TestRequired{Name: "", Email: "john@example.com", Age: 25},
			wantErr: true,
		},
		{
			name:    "zero int fails",
			input:   TestRequired{Name: "John", Email: "john@example.com", Age: 0},
			wantErr: true,
		},
		{
			name:    "all empty fails",
			input:   TestRequired{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequiredOnPointer(t *testing.T) {
	type TestPtr struct {
		Name *string `validate:"required"`
		Age  *int    `validate:"required"`
	}

	tests := []struct {
		name    string
		input   TestPtr
		wantErr bool
	}{
		{
			name: "valid pointers",
			input: func() TestPtr {
				name, age := "John", 25
				return TestPtr{Name: &name, Age: &age}
			}(),
			wantErr: false,
		},
		{
			name:    "nil pointer fails",
			input:   TestPtr{},
			wantErr: true,
		},
		{
			name: "one nil fails",
			input: func() TestPtr {
				name := "John"
				return TestPtr{Name: &name}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequiredOnSlice(t *testing.T) {
	type TestSlice struct {
		Tags []string `validate:"required"`
	}

	tests := []struct {
		name    string
		input   TestSlice
		wantErr bool
	}{
		{
			name:    "non-empty slice",
			input:   TestSlice{Tags: []string{"a", "b"}},
			wantErr: false,
		},
		{
			name:    "empty slice fails",
			input:   TestSlice{Tags: []string{}},
			wantErr: true,
		},
		{
			name:    "nil slice fails",
			input:   TestSlice{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequiredOnMap(t *testing.T) {
	type TestMap struct {
		Data map[string]int `validate:"required"`
	}

	tests := []struct {
		name    string
		input   TestMap
		wantErr bool
	}{
		{
			name:    "non-empty map",
			input:   TestMap{Data: map[string]int{"a": 1}},
			wantErr: false,
		},
		{
			name:    "empty map fails",
			input:   TestMap{Data: map[string]int{}},
			wantErr: true,
		},
		{
			name:    "nil map fails",
			input:   TestMap{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOmitEmptyValidator(t *testing.T) {
	type TestOmitEmpty struct {
		Name  string `validate:"omitempty,min=2"`
		Email string `validate:"omitempty,email"`
		Age   int    `validate:"omitempty,min=13"`
	}

	tests := []struct {
		name    string
		input   TestOmitEmpty
		wantErr bool
	}{
		{
			name:    "all empty is valid",
			input:   TestOmitEmpty{},
			wantErr: false,
		},
		{
			name:    "valid values",
			input:   TestOmitEmpty{Name: "John", Email: "john@example.com", Age: 25},
			wantErr: false,
		},
		{
			name:    "name too short",
			input:   TestOmitEmpty{Name: "J", Email: "john@example.com", Age: 25},
			wantErr: true,
		},
		{
			name:    "invalid email",
			input:   TestOmitEmpty{Name: "John", Email: "invalid", Age: 25},
			wantErr: true,
		},
		{
			name:    "age too small",
			input:   TestOmitEmpty{Name: "John", Email: "john@example.com", Age: 10},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOmitEmptyOnPointer(t *testing.T) {
	type TestPtr struct {
		Name *string `validate:"omitempty,min=2"`
		Age  *int    `validate:"omitempty,min=13"`
	}

	tests := []struct {
		name    string
		input   TestPtr
		wantErr bool
	}{
		{
			name:    "nil pointers valid",
			input:   TestPtr{},
			wantErr: false,
		},
		{
			name: "valid values",
			input: func() TestPtr {
				name, age := "John", 25
				return TestPtr{Name: &name, Age: &age}
			}(),
			wantErr: false,
		},
		{
			name: "too short fails",
			input: func() TestPtr {
				name := "J"
				return TestPtr{Name: &name}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequiredWithOmitEmpty(t *testing.T) {
	type TestMixed struct {
		RequiredName    string `validate:"required"`
		OptionalComment string `validate:"omitempty,min=5"`
	}

	tests := []struct {
		name    string
		input   TestMixed
		wantErr bool
	}{
		{
			name:    "required present, optional empty",
			input:   TestMixed{RequiredName: "John"},
			wantErr: false,
		},
		{
			name:    "required missing",
			input:   TestMixed{OptionalComment: "hello"},
			wantErr: true,
		},
		{
			name:    "both present and valid",
			input:   TestMixed{RequiredName: "John", OptionalComment: "hello world"},
			wantErr: false,
		},
		{
			name:    "optional too short",
			input:   TestMixed{RequiredName: "John", OptionalComment: "hi"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMinMaxValidator(t *testing.T) {
	type TestMinMax struct {
		Name  string  `validate:"min=2,max=50"`
		Age   int     `validate:"min=13,max=120"`
		Score uint    `validate:"min=0,max=100"`
		Rate  float64 `validate:"min=0.0,max=1.0"`
	}

	tests := []struct {
		name     string
		input    TestMinMax
		wantErr  bool
		errField string
	}{
		{
			name:    "all valid",
			input:   TestMinMax{Name: "John", Age: 25, Score: 85, Rate: 0.5},
			wantErr: false,
		},
		{
			name:     "string too short",
			input:    TestMinMax{Name: "J", Age: 25, Score: 85, Rate: 0.5},
			wantErr:  true,
			errField: "Name",
		},
		{
			name:     "string too long",
			input:    TestMinMax{Name: "a very long name that exceeds the maximum allowed length of fifty characters", Age: 25, Score: 85, Rate: 0.5},
			wantErr:  true,
			errField: "Name",
		},
		{
			name:     "int too small",
			input:    TestMinMax{Name: "John", Age: 10, Score: 85, Rate: 0.5},
			wantErr:  true,
			errField: "Age",
		},
		{
			name:     "int too large",
			input:    TestMinMax{Name: "John", Age: 150, Score: 85, Rate: 0.5},
			wantErr:  true,
			errField: "Age",
		},
		{
			name:     "uint exceeds max",
			input:    TestMinMax{Name: "John", Age: 25, Score: 101, Rate: 0.5},
			wantErr:  true,
			errField: "Score",
		},
		{
			name:     "float exceeds max",
			input:    TestMinMax{Name: "John", Age: 25, Score: 85, Rate: 1.5},
			wantErr:  true,
			errField: "Rate",
		},
		{
			name:     "float below min",
			input:    TestMinMax{Name: "John", Age: 25, Score: 85, Rate: -0.5},
			wantErr:  true,
			errField: "Rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
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
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLenValidator(t *testing.T) {
	type TestLen struct {
		Name  string   `validate:"len=5"`
		Items []string `validate:"len=3"`
		Array [3]int   `validate:"len=3"`
	}

	tests := []struct {
		name    string
		input   TestLen
		wantErr bool
	}{
		{
			name:    "all valid",
			input:   TestLen{Name: "hello", Items: []string{"a", "b", "c"}, Array: [3]int{1, 2, 3}},
			wantErr: false,
		},
		{
			name:    "string wrong length",
			input:   TestLen{Name: "hi", Items: []string{"a", "b", "c"}, Array: [3]int{1, 2, 3}},
			wantErr: true,
		},
		{
			name:    "slice wrong count",
			input:   TestLen{Name: "hello", Items: []string{"a", "b"}, Array: [3]int{1, 2, 3}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLenOnUnsupportedType(t *testing.T) {
	type TestLenInt struct {
		Value int `validate:"len=5"`
	}
	input := TestLenInt{Value: 10}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for len on int")
	}
}

func TestLenOnMap(t *testing.T) {
	type TestLenMap struct {
		Items map[string]int `validate:"len=2"`
	}

	tests := []struct {
		name    string
		items   map[string]int
		wantErr bool
	}{
		{"correct length", map[string]int{"a": 1, "b": 2}, false},
		{"wrong length", map[string]int{"a": 1}, true},
		{"empty map", map[string]int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestLenMap{Items: tt.items}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %v", tt.items)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %v: %v", tt.items, err)
			}
		})
	}
}

func TestEqValidator(t *testing.T) {
	type TestEq struct {
		Status string  `validate:"eq=active"`
		Count  int     `validate:"eq=5"`
		Rate   float64 `validate:"eq=3.14"`
	}

	tests := []struct {
		name    string
		input   TestEq
		wantErr bool
	}{
		{"string match", TestEq{Status: "active", Count: 5, Rate: 3.14}, false},
		{"string no match", TestEq{Status: "inactive", Count: 5, Rate: 3.14}, true},
		{"int match", TestEq{Status: "active", Count: 5, Rate: 3.14}, false},
		{"int no match", TestEq{Status: "active", Count: 3, Rate: 3.14}, true},
		{"float match", TestEq{Status: "active", Count: 5, Rate: 3.14}, false},
		{"float no match", TestEq{Status: "active", Count: 5, Rate: 2.71}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEqInvalidParam(t *testing.T) {
	tests := []struct {
		name  string
		setup func() any
	}{
		{
			name: "eq uint with invalid param",
			setup: func() any {
				type Test struct {
					Value uint `validate:"eq=abc"`
				}
				return &Test{Value: 5}
			},
		},
		{
			name: "eq float with invalid param",
			setup: func() any {
				type Test struct {
					Value float64 `validate:"eq=xyz"`
				}
				return &Test{Value: 3.14}
			},
		},
		{
			name: "eq on unsupported type",
			setup: func() any {
				type Test struct {
					Value bool `validate:"eq=true"`
				}
				return &Test{Value: true}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			err := NewValidator().Struct(input)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestNeValidator(t *testing.T) {
	type TestNe struct {
		Status string  `validate:"ne=deleted"`
		Count  int     `validate:"ne=0"`
		Rate   float64 `validate:"ne=1.0"`
	}

	tests := []struct {
		name    string
		input   TestNe
		wantErr bool
	}{
		{"string different", TestNe{Status: "active", Count: 5, Rate: 0.5}, false},
		{"string equal", TestNe{Status: "deleted", Count: 5, Rate: 0.5}, true},
		{"int different", TestNe{Status: "active", Count: 5, Rate: 0.5}, false},
		{"int equal", TestNe{Status: "active", Count: 0, Rate: 0.5}, true},
		{"float different", TestNe{Status: "active", Count: 5, Rate: 0.5}, false},
		{"float equal", TestNe{Status: "active", Count: 5, Rate: 1.0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNeInvalidParam(t *testing.T) {
	tests := []struct {
		name  string
		setup func() any
	}{
		{
			name: "ne uint with invalid param",
			setup: func() any {
				type Test struct {
					Value uint `validate:"ne=abc"`
				}
				return &Test{Value: 5}
			},
		},
		{
			name: "ne float with invalid param",
			setup: func() any {
				type Test struct {
					Value float64 `validate:"ne=xyz"`
				}
				return &Test{Value: 3.14}
			},
		},
		{
			name: "ne on unsupported type",
			setup: func() any {
				type Test struct {
					Value bool `validate:"ne=true"`
				}
				return &Test{Value: true}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			err := NewValidator().Struct(input)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestGtLtValidator(t *testing.T) {
	type TestGtLt struct {
		Score int     `validate:"gt=0,lt=100"`
		Rate  float64 `validate:"gt=0.0,lt=1.0"`
	}

	tests := []struct {
		name    string
		input   TestGtLt
		wantErr bool
	}{
		{"valid values", TestGtLt{Score: 50, Rate: 0.5}, false},
		{"score too low", TestGtLt{Score: 0, Rate: 0.5}, true},
		{"score too high", TestGtLt{Score: 100, Rate: 0.5}, true},
		{"rate too low", TestGtLt{Score: 50, Rate: 0.0}, true},
		{"rate too high", TestGtLt{Score: 50, Rate: 1.0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGteLteValidator(t *testing.T) {
	type TestGteLte struct {
		Score int     `validate:"gte=0,lte=100"`
		Rate  float64 `validate:"gte=0.0,lte=1.0"`
	}

	tests := []struct {
		name    string
		input   TestGteLte
		wantErr bool
	}{
		{"valid values", TestGteLte{Score: 50, Rate: 0.5}, false},
		{"score at min", TestGteLte{Score: 0, Rate: 0.5}, false},
		{"score at max", TestGteLte{Score: 100, Rate: 0.5}, false},
		{"rate at min", TestGteLte{Score: 50, Rate: 0.0}, false},
		{"rate at max", TestGteLte{Score: 50, Rate: 1.0}, false},
		{"score below min", TestGteLte{Score: -1, Rate: 0.5}, true},
		{"score above max", TestGteLte{Score: 101, Rate: 0.5}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGtLtInvalidParamAndUnsupportedType(t *testing.T) {
	tests := []struct {
		name  string
		setup func() any
	}{
		{
			name: "gt with invalid param",
			setup: func() any {
				type Test struct {
					Value int `validate:"gt=abc"`
				}
				return &Test{Value: 5}
			},
		},
		{
			name: "lt with invalid param",
			setup: func() any {
				type Test struct {
					Value int `validate:"lt=xyz"`
				}
				return &Test{Value: 5}
			},
		},
		{
			name: "gt on unsupported type",
			setup: func() any {
				type Test struct {
					Value bool `validate:"gt=1"`
				}
				return &Test{Value: true}
			},
		},
		{
			name: "lt on unsupported type",
			setup: func() any {
				type Test struct {
					Value string `validate:"lt=1"`
				}
				return &Test{Value: "abc"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			err := NewValidator().Struct(input)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestGteLteInvalidParamAndUnsupportedType(t *testing.T) {
	tests := []struct {
		name  string
		setup func() any
	}{
		{
			name: "gte with invalid param",
			setup: func() any {
				type Test struct {
					Value int `validate:"gte=abc"`
				}
				return &Test{Value: 5}
			},
		},
		{
			name: "lte with invalid param",
			setup: func() any {
				type Test struct {
					Value int `validate:"lte=xyz"`
				}
				return &Test{Value: 5}
			},
		},
		{
			name: "gte on unsupported type",
			setup: func() any {
				type Test struct {
					Value bool `validate:"gte=true"`
				}
				return &Test{Value: true}
			},
		},
		{
			name: "lte on unsupported type",
			setup: func() any {
				type Test struct {
					Value bool `validate:"lte=false"`
				}
				return &Test{Value: false}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			err := NewValidator().Struct(input)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestEmailValidator(t *testing.T) {
	type TestEmail struct {
		Email string `validate:"email"`
	}

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"empty", "", false},
		{"simple valid", "a@b.co", false},
		{"standard valid", "test@example.com", false},
		{"with dots", "first.last@example.com", false},
		{"with plus", "test+tag@example.com", false},
		{"no @", "testexample.com", true},
		{"no domain", "test@", true},
		{"no local", "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestEmail{Email: tt.email}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.email)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.email, err)
			}
		})
	}
}

func TestEmailOnNonString(t *testing.T) {
	type TestEmailInt struct {
		Value int `validate:"email"`
	}
	input := TestEmailInt{Value: 123}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for email validator on non-string type")
	}
}

func TestAlphaValidator(t *testing.T) {
	type TestAlpha struct {
		Name string `validate:"alpha"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"letters only", "John", false},
		{"with space", "John Doe", true},
		{"with number", "John123", true},
		{"unicode", "日本語", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestAlpha{Name: tt.value}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.value, err)
			}
		})
	}
}

func TestAlphaOnNonString(t *testing.T) {
	type TestAlphaInt struct {
		Value int `validate:"alpha"`
	}
	input := TestAlphaInt{Value: 123}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for alpha validator on non-string type")
	}
}

func TestAlphanumValidator(t *testing.T) {
	type TestAlphanum struct {
		Code string `validate:"alphanum"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"letters and numbers", "ABC123", false},
		{"with hyphen", "ABC-123", true},
		{"with space", "ABC 123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestAlphanum{Code: tt.value}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.value, err)
			}
		})
	}
}

func TestAlphanumOnNonString(t *testing.T) {
	type TestAlphanumInt struct {
		Value int `validate:"alphanum"`
	}
	input := TestAlphanumInt{Value: 123}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for alphanum validator on non-string type")
	}
}

func TestNumericValidator(t *testing.T) {
	type TestNumeric struct {
		Value string `validate:"numeric"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"numbers only", "12345", false},
		{"with letter", "123a45", true},
		{"with space", "123 45", true},
		{"decimal", "123.45", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestNumeric{Value: tt.value}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.value, err)
			}
		})
	}
}

func TestNumericOnNonString(t *testing.T) {
	type TestNumericInt struct {
		Value int `validate:"numeric"`
	}
	input := TestNumericInt{Value: 123}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for numeric validator on non-string type")
	}
}

func TestOneOfValidator(t *testing.T) {
	type TestOneOf struct {
		Status string `validate:"oneof=active inactive pending"`
		Level  int    `validate:"oneof=1 2 3"`
	}

	tests := []struct {
		name    string
		input   TestOneOf
		wantErr bool
	}{
		{"valid status", TestOneOf{Status: "active", Level: 1}, false},
		{"invalid status", TestOneOf{Status: "deleted", Level: 1}, true},
		{"valid level", TestOneOf{Status: "active", Level: 2}, false},
		{"invalid level", TestOneOf{Status: "active", Level: 5}, true},
		{"empty string valid", TestOneOf{Status: "", Level: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestOneOfUint(t *testing.T) {
	type TestOneOfUint struct {
		Code uint `validate:"oneof=1 2 3"`
	}

	tests := []struct {
		name    string
		input   uint
		wantErr bool
	}{
		{"valid uint", 1, false},
		{"valid uint 2", 2, false},
		{"invalid uint", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestOneOfUint{Code: tt.input}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %d", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %d: %v", tt.input, err)
			}
		})
	}
}

func TestOneOfEmptyOptions(t *testing.T) {
	type TestOneOfEmpty struct {
		Value string `validate:"oneof="`
	}
	input := TestOneOfEmpty{Value: "anything"}
	err := NewValidator().Struct(&input)
	if err != nil {
		t.Errorf("expected no error with empty options, got %v", err)
	}
}

func TestOneOfUnsupportedType(t *testing.T) {
	type TestOneOfFloat struct {
		Value float64 `validate:"oneof=1.5 2.5"`
	}
	input := TestOneOfFloat{Value: 1.5}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for oneof on float64")
	}
}

func TestUUIDValidator(t *testing.T) {
	type TestUUID struct {
		ID string `validate:"uuid"`
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid v1", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", false},
		{"uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"no dashes", "550e8400e29b41d4a716446655440000", true},
		{"too short", "550e8400-e29b-41d4-a716-44665544000", true},
		{"invalid chars", "550e8400-e29b-41d4-XXXX-446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestUUID{ID: tt.id}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.id)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.id, err)
			}
		})
	}
}

func TestUUIDOnNonString(t *testing.T) {
	type TestUUIDInt struct {
		Value int `validate:"uuid"`
	}
	input := TestUUIDInt{Value: 123}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for uuid validator on non-string type")
	}
}

func TestDatetimeValidator(t *testing.T) {
	type TestDatetime struct {
		Date string `validate:"datetime=2006-01-02"`
	}

	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid date", "2024-01-15", false},
		{"wrong format", "15-01-2024", true},
		{"invalid date", "2024-13-45", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestDatetime{Date: tt.date}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.date)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.date, err)
			}
		})
	}
}

func TestDatetimeOnNonString(t *testing.T) {
	type TestDatetimeInt struct {
		Value int `validate:"datetime"`
	}
	input := TestDatetimeInt{Value: 123}
	err := NewValidator().Struct(&input)
	if err == nil {
		t.Error("expected error for datetime validator on non-string type")
	}
}

func TestDatetimeDefaultFormat(t *testing.T) {
	type TestDatetimeDefault struct {
		Timestamp string `validate:"datetime"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid RFC3339", "2024-01-15T10:30:00Z", false},
		{"valid RFC3339 with offset", "2024-01-15T10:30:00+05:00", false},
		{"invalid RFC3339", "2024-01-15 10:30:00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestDatetimeDefault{Timestamp: tt.value}
			err := NewValidator().Struct(&input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.value, err)
			}
		})
	}
}

func TestCoreValidatorsCombined(t *testing.T) {
	type TestUser struct {
		Name  string `validate:"required,min=2,max=50"`
		Email string `validate:"required,email"`
		Age   int    `validate:"min=13,max=120"`
	}

	tests := []struct {
		name     string
		input    TestUser
		wantErr  bool
		errField string
	}{
		{
			name:    "valid user",
			input:   TestUser{Name: "John", Email: "john@example.com", Age: 25},
			wantErr: false,
		},
		{
			name:     "missing name",
			input:    TestUser{Email: "john@example.com", Age: 25},
			wantErr:  true,
			errField: "Name",
		},
		{
			name:     "name too short",
			input:    TestUser{Name: "J", Email: "john@example.com", Age: 25},
			wantErr:  true,
			errField: "Name",
		},
		{
			name:     "age too small",
			input:    TestUser{Name: "John", Email: "john@example.com", Age: 10},
			wantErr:  true,
			errField: "Age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
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
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestTimeFieldSkipped tests that time.Time fields are skipped during recursive validation
func TestTimeFieldSkipped(t *testing.T) {
	type TestWithTime struct {
		Name      string    `validate:"required"`
		CreatedAt time.Time `validate:"-"` // time.Time should be skipped
	}

	tests := []struct {
		name    string
		input   TestWithTime
		wantErr bool
	}{
		{
			name:    "valid with time field",
			input:   TestWithTime{Name: "John", CreatedAt: time.Now()},
			wantErr: false,
		},
		{
			name:    "missing name fails",
			input:   TestWithTime{Name: "", CreatedAt: time.Now()},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestUnexportedFields tests that unexported fields are skipped during validation
func TestUnexportedFields(t *testing.T) {
	type TestUnexported struct {
		Name            string `validate:"required"`
		unexportedField string `validate:"required"` // unexported, should be skipped
		Age             int    `validate:"min=0"`
	}

	// Suppress linter warning about unused field - we intentionally don't use it
	_ = TestUnexported{unexportedField: "ignored"}.unexportedField

	// Unexported fields should not cause validation errors even with invalid data
	tests := []struct {
		name    string
		input   TestUnexported
		wantErr bool
	}{
		{
			name:    "valid with unexported field",
			input:   TestUnexported{Name: "John", Age: 25},
			wantErr: false,
		},
		{
			name:    "exported fails but unexported ignored",
			input:   TestUnexported{Name: "", Age: 25}, // Name is empty but required
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestSliceOfPointerToStruct tests validation of slices containing pointer-to-struct elements
func TestSliceOfPointerToStruct(t *testing.T) {
	type Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
	}

	type Person struct {
		Name      string     `validate:"required"`
		Addresses []*Address // no validate tag - should still recurse into slice elements
	}

	tests := []struct {
		name    string
		input   Person
		wantErr bool
	}{
		{
			name:    "all valid pointers",
			input:   Person{Name: "John", Addresses: []*Address{{Street: "123 Main", City: "NYC"}}},
			wantErr: false,
		},
		{
			name:    "nil pointer in slice",
			input:   Person{Name: "John", Addresses: []*Address{nil}},
			wantErr: false, // nil pointers are skipped
		},
		{
			name:    "invalid nested struct",
			input:   Person{Name: "John", Addresses: []*Address{{Street: "", City: "NYC"}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestMapOfPointerToStruct tests validation of maps containing pointer-to-struct values
func TestMapOfPointerToStruct(t *testing.T) {
	type Config struct {
		Value string `validate:"required"`
	}

	type App struct {
		Name    string             `validate:"required"`
		Configs map[string]*Config // no validate tag - should still recurse into map values
	}

	tests := []struct {
		name    string
		input   App
		wantErr bool
	}{
		{
			name:    "all valid pointers in map",
			input:   App{Name: "MyApp", Configs: map[string]*Config{"db": {Value: "localhost"}}},
			wantErr: false,
		},
		{
			name:    "nil pointer value in map",
			input:   App{Name: "MyApp", Configs: map[string]*Config{"db": nil}},
			wantErr: false, // nil pointers are skipped
		},
		{
			name:    "invalid nested struct in map",
			input:   App{Name: "MyApp", Configs: map[string]*Config{"db": {Value: ""}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestValidationErrors_Error tests the Error() method
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

// TestValidationErrors_HasErrors tests the HasErrors() method
func TestValidationErrors_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   bool
	}{
		{
			name:   "empty errors",
			errors: ValidationErrors{},
			want:   false,
		},
		{
			name:   "nil errors",
			errors: nil,
			want:   false,
		},
		{
			name:   "has errors",
			errors: ValidationErrors{"field": {"error"}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.HasErrors()
			if got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValidationErrors_ValidationErrors tests the ValidationErrors() method
func TestValidationErrors_ValidationErrors(t *testing.T) {
	errors := ValidationErrors{"field": {"error1", "error2"}}
	got := errors.ValidationErrors()

	if len(got) != 1 {
		t.Errorf("ValidationErrors() returned %d entries, want 1", len(got))
	}

	if errs, ok := got["field"]; !ok || len(errs) != 2 {
		t.Errorf("ValidationErrors() = %v, want map with field having 2 errors", got)
	}
}

// TestRegister tests custom validator registration
func TestRegister(t *testing.T) {
	type TestCustom struct {
		Value string `validate:"custom"`
	}

	v := NewValidator()

	// Register a custom validator
	customCalled := false
	v.Register("custom", func(value reflect.Value, param string) error {
		customCalled = true
		if value.String() != "valid" {
			return errors.New("must be 'valid'")
		}
		return nil
	})

	// Test valid case
	valid := TestCustom{Value: "valid"}
	if err := v.Struct(&valid); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !customCalled {
		t.Error("custom validator was not called")
	}

	// Test invalid case
	customCalled = false
	invalid := TestCustom{Value: "invalid"}
	if err := v.Struct(&invalid); err == nil {
		t.Error("expected error, got nil")
	}
}

// TestRegister_MultipleCustomValidators tests registering multiple custom validators
func TestRegister_MultipleCustomValidators(t *testing.T) {
	type TestMultiple struct {
		A string `validate:"customA"`
		B string `validate:"customB"`
	}

	v := NewValidator()
	v.Register("customA", func(value reflect.Value, param string) error {
		return nil
	})
	v.Register("customB", func(value reflect.Value, param string) error {
		return errors.New("customB error")
	})

	input := TestMultiple{A: "a", B: "b"}
	err := v.Struct(&input)
	if err == nil {
		t.Error("expected error from customB")
	}
}

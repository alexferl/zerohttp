package validator

import (
	"testing"
)

func TestContainsValidator(t *testing.T) {
	type TestContains struct {
		Value string `validate:"contains=test"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"contains substring", "this is a test", false},
		{"does not contain", "hello world", true},
		{"partial match", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestContains{Value: tt.value}
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

func TestContainsOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"contains=test"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for contains on non-string")
	}
}

func TestStartsWithValidator(t *testing.T) {
	type TestStartsWith struct {
		Value string `validate:"startswith=hello"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"starts with prefix", "hello there", false},
		{"exact match", "hello", false},
		{"wrong start", "goodbye", true},
		{"contains but not starts", "say hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestStartsWith{Value: tt.value}
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

func TestStartsWithOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"startswith=hello"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for startswith on non-string")
	}
}

func TestEndsWithValidator(t *testing.T) {
	type TestEndsWith struct {
		Value string `validate:"endswith=world"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"ends with suffix", "hello world", false},
		{"exact match", "world", false},
		{"wrong end", "hello planet", true},
		{"contains but not ends", "worldly", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestEndsWith{Value: tt.value}
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

func TestEndsWithOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"endswith=world"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for endswith on non-string")
	}
}

func TestExcludesValidator(t *testing.T) {
	type TestExcludes struct {
		Value string `validate:"excludes=badword"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid content", "hello world", false},
		{"contains excluded", "this has badword in it", true},
		{"excluded at end", "ends with badword", true},
		{"excluded at start", "badword at start", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestExcludes{Value: tt.value}
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

func TestExcludesOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"excludes=test"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for excludes on non-string")
	}
}

func TestLowercaseValidator(t *testing.T) {
	type TestLowercase struct {
		Value string `validate:"lowercase"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"all lowercase", "hello world", false},
		{"mixed case", "Hello World", true},
		{"all uppercase", "HELLO", true},
		{"with numbers", "hello123", false},
		{"with unicode lowercase", "café", false},
		{"unicode uppercase fails", "CAFÉ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestLowercase{Value: tt.value}
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

func TestLowercaseOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"lowercase"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for lowercase on non-string")
	}
}

func TestUppercaseValidator(t *testing.T) {
	type TestUppercase struct {
		Value string `validate:"uppercase"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"all uppercase", "HELLO WORLD", false},
		{"mixed case", "Hello World", true},
		{"all lowercase", "hello", true},
		{"with numbers", "HELLO123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestUppercase{Value: tt.value}
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

func TestUppercaseOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"uppercase"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for uppercase on non-string")
	}
}

func TestASCIIValidator(t *testing.T) {
	type TestASCII struct {
		Value string `validate:"ascii"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid ascii", "hello world 123", false},
		{"newline allowed", "hello\nworld", false},
		{"tab allowed", "hello\tworld", false},
		{"unicode fails", "日本語", true},
		{"mixed unicode fails", "hello 日本", true},
		{"accented char fails", "café", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestASCII{Value: tt.value}
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

func TestASCIOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"ascii"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for ascii on non-string")
	}
}

func TestPrintASCIIValidator(t *testing.T) {
	type TestPrintASCII struct {
		Value string `validate:"printascii"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid printable", "hello world!", false},
		{"newline fails", "hello\nworld", true},
		{"tab fails", "hello\tworld", true},
		{"unicode fails", "日本語", true},
		{"mixed printable", "hello world 123!@#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestPrintASCII{Value: tt.value}
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

func TestPrintASCIOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"printascii"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for printascii on non-string")
	}
}

func TestBooleanValidator(t *testing.T) {
	type TestBoolean struct {
		Value string `validate:"boolean"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"true lowercase", "true", false},
		{"false lowercase", "false", false},
		{"true uppercase", "TRUE", false},
		{"false uppercase", "FALSE", false},
		{"one", "1", false},
		{"zero", "0", false},
		{"yes lowercase", "yes", false},
		{"no lowercase", "no", false},
		{"yes uppercase", "YES", false},
		{"no uppercase", "NO", false},
		{"on lowercase", "on", false},
		{"off lowercase", "off", false},
		{"on uppercase", "ON", false},
		{"off uppercase", "OFF", false},
		{"invalid word", "maybe", true},
		{"invalid number", "2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&TestBoolean{Value: tt.value})
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.value, err)
			}
		})
	}
}

func TestBooleanOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"boolean"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for boolean on non-string")
	}
}

func TestJSONValidator(t *testing.T) {
	type TestJSON struct {
		Value string `validate:"json"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid object", `{"key":"value"}`, false},
		{"valid array", `[1,2,3]`, false},
		{"valid string", `"hello"`, false},
		{"valid number", `42`, false},
		{"valid boolean", `true`, false},
		{"valid null", `null`, false},
		{"invalid syntax", "not valid json", true},
		{"invalid trailing comma", `{"a":1,}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidator().Struct(&TestJSON{Value: tt.value})
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.value, err)
			}
		})
	}
}

func TestJSONOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"json"`
	}
	input := Test{Value: 123}
	if err := NewValidator().Struct(&input); err == nil {
		t.Error("expected error for json on non-string")
	}
}

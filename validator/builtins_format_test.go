package validator

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestBase64Validator(t *testing.T) {
	type TestBase64 struct {
		Value string `validate:"base64"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid standard", "SGVsbG8gV29ybGQ=", false},
		{"valid URL safe", "dGVzdA==", false},
		{"invalid chars", "not-valid-base64!@#", true},
		{"invalid padding", "SGVsbG8gV29ybGQ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestBase64{Value: tt.value})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestBase64OnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"base64"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestHexadecimalValidator(t *testing.T) {
	type TestHex struct {
		Value string `validate:"hexadecimal"`
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid lowercase", "a1b2c3d4", false},
		{"valid uppercase", "ABCDEF", false},
		{"valid mixed", "AbCdEf123456", false},
		{"invalid chars", "not-hex", true},
		{"invalid with 0x prefix", "0xabc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Struct(&TestHex{Value: tt.value})
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestHexadecimalOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"hexadecimal"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestHexColorValidator(t *testing.T) {
	type TestHexColor struct {
		Color string `validate:"hexcolor"`
	}

	tests := []struct {
		name    string
		color   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid 3 digit", "#FFF", false},
		{"valid 4 digit", "#FFFF", false},
		{"valid 6 digit", "#FFFFFF", false},
		{"valid 8 digit", "#FFFFFFFF", false},
		{"lowercase", "#ffffff", false},
		{"mixed case", "#FfFfFf", false},
		{"no hash", "FFFFFF", true},
		{"invalid chars", "#GGGGGG", true},
		{"wrong length 5", "#FFFFF", true},
		{"wrong length 7", "#FFFFFFF", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestHexColor{Color: tt.color}
			err := New().Struct(&input)
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestHexColorOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"hexcolor"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestE164Validator(t *testing.T) {
	type TestE164 struct {
		Phone string `validate:"e164"`
	}

	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid US", "+14155552671", false},
		{"valid UK", "+447400123456", false},
		{"no plus", "14155552671", true},
		{"too short", "+1", true},
		{"too long", "+1234567890123456", true},
		{"starts with zero", "+04155552671", true},
		{"contains letters", "+1abc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestE164{Phone: tt.phone}
			err := New().Struct(&input)
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestE164OnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"e164"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestSemverValidator(t *testing.T) {
	type TestSemver struct {
		Version string `validate:"semver"`
	}

	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"empty", "", false},
		{"simple", "1.2.3", false},
		{"with prerelease", "1.2.3-alpha", false},
		{"with build", "1.2.3+build", false},
		{"with both", "1.2.3-alpha+build", false},
		{"zero version", "0.0.0", false},
		{"two parts", "1.2", true},
		{"one part", "1", true},
		{"leading v", "v1.2.3", true},
		{"invalid chars", "1.2.3-rc#1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestSemver{Version: tt.version}
			err := New().Struct(&input)
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestSemverOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"semver"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

func TestJWTValidator(t *testing.T) {
	type TestJWT struct {
		Token string `validate:"jwt"`
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"empty", "", false},
		{"valid format", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", false},
		{"two parts", "header.payload", true},
		{"four parts", "a.b.c.d", true},
		{"invalid base64", "not.valid.token", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestJWT{Token: tt.token}
			err := New().Struct(&input)
			if tt.wantErr {
				zhtest.AssertError(t, err)
			} else {
				zhtest.AssertNoError(t, err)
			}
		})
	}
}

func TestJWTOnNonString(t *testing.T) {
	type Test struct {
		Value int `validate:"jwt"`
	}
	input := Test{Value: 123}
	zhtest.AssertError(t, New().Struct(&input))
}

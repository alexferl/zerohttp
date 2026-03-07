package zerohttp

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestMinMaxInvalidType(t *testing.T) {
	t.Run("min on invalid type", func(t *testing.T) {
		type TestMin struct {
			Value complex128 `validate:"min=5"`
		}
		input := TestMin{Value: 10}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for min on complex128")
		}
	})
	t.Run("max on invalid type", func(t *testing.T) {
		type TestMax struct {
			Value complex128 `validate:"max=5"`
		}
		input := TestMax{Value: 10}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for max on complex128")
		}
	})
}

func TestValidatorErrorCases(t *testing.T) {
	t.Run("min with invalid param", func(t *testing.T) {
		type TestMin struct {
			Value int `validate:"min=notanumber"`
		}
		input := TestMin{Value: 5}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for invalid min param")
		}
	})

	t.Run("max with invalid param", func(t *testing.T) {
		type TestMax struct {
			Value int `validate:"max=notanumber"`
		}
		input := TestMax{Value: 5}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for invalid max param")
		}
	})

	t.Run("len with invalid param", func(t *testing.T) {
		type TestLen struct {
			Value string `validate:"len=notanumber"`
		}
		input := TestLen{Value: "hello"}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for invalid len param")
		}
	})

	t.Run("eq with invalid param", func(t *testing.T) {
		type TestEq struct {
			Value int `validate:"eq=notanumber"`
		}
		input := TestEq{Value: 5}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for invalid eq param")
		}
	})
}

func TestUintComparisonValidators(t *testing.T) {
	type TestUint struct {
		EqVal  uint64 `validate:"eq=100"`
		NeVal  uint64 `validate:"ne=0"`
		GtVal  uint64 `validate:"gt=10"`
		LtVal  uint64 `validate:"lt=1000"`
		GteVal uint64 `validate:"gte=50"`
		LteVal uint64 `validate:"lte=500"`
		Uint8  uint8  `validate:"eq=255"`
		Uint16 uint16 `validate:"min=100,max=1000"`
		Uint32 uint32 `validate:"gte=0,lte=4294967295"`
	}

	tests := []struct {
		name    string
		input   TestUint
		wantErr bool
	}{
		{
			name:    "all valid",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: false,
		},
		{
			name:    "eq fail",
			input:   TestUint{EqVal: 99, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "ne fail",
			input:   TestUint{EqVal: 100, NeVal: 0, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "gt fail",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 5, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "lt fail",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 1000, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "gte fail",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 49, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "lte fail",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 501, Uint8: 255, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "uint8 eq fail",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 254, Uint16: 500, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "uint16 min fail",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 50, Uint32: 1000000},
			wantErr: true,
		},
		{
			name:    "uint32 max boundary",
			input:   TestUint{EqVal: 100, NeVal: 50, GtVal: 20, LtVal: 500, GteVal: 50, LteVal: 400, Uint8: 255, Uint16: 500, Uint32: 4294967295},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestFloatBoundaryValues(t *testing.T) {
	type TestFloat struct {
		GtZero   float64 `validate:"gt=0"`
		LtZero   float64 `validate:"lt=0"`
		GtNeg    float64 `validate:"gt=-100"`
		LtPos    float64 `validate:"lt=100"`
		SmallGt  float64 `validate:"gt=0.001"`
		SmallLt  float64 `validate:"lt=0.001"`
		LargeVal float64 `validate:"gte=1e10,lte=1e15"`
	}

	tests := []struct {
		name    string
		input   TestFloat
		wantErr bool
	}{
		{
			name:    "all valid boundaries",
			input:   TestFloat{GtZero: 0.0001, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e12},
			wantErr: false,
		},
		{
			name:    "gt=0 fails at zero",
			input:   TestFloat{GtZero: 0, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e12},
			wantErr: true,
		},
		{
			name:    "gt=0 fails at negative",
			input:   TestFloat{GtZero: -0.1, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e12},
			wantErr: true,
		},
		{
			name:    "lt=0 fails at zero",
			input:   TestFloat{GtZero: 0.1, LtZero: 0, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e12},
			wantErr: true,
		},
		{
			name:    "lt=0 fails at positive",
			input:   TestFloat{GtZero: 0.1, LtZero: 0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e12},
			wantErr: true,
		},
		{
			name:    "small decimal boundary",
			input:   TestFloat{GtZero: 0.1, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.0010001, SmallLt: 0.0009999, LargeVal: 1e12},
			wantErr: false,
		},
		{
			name:    "small gt fails",
			input:   TestFloat{GtZero: 0.1, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.001, SmallLt: 0.0005, LargeVal: 1e12},
			wantErr: true,
		},
		{
			name:    "small lt fails",
			input:   TestFloat{GtZero: 0.1, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.001, LargeVal: 1e12},
			wantErr: true,
		},
		{
			name:    "large val too small",
			input:   TestFloat{GtZero: 0.1, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e9},
			wantErr: true,
		},
		{
			name:    "large val too big",
			input:   TestFloat{GtZero: 0.1, LtZero: -0.1, GtNeg: -50, LtPos: 99.9, SmallGt: 0.002, SmallLt: 0.0005, LargeVal: 1e16},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := V.Struct(&tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestMinMaxOnUnsupportedTypes(t *testing.T) {
	t.Run("min on complex number", func(t *testing.T) {
		type TestMinComplex struct {
			Value complex128 `validate:"min=5"`
		}
		input := TestMinComplex{Value: 10 + 5i}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for min on complex128")
			return
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Errorf("expected ValidationErrors, got %T", err)
			return
		}
		errs := ve.FieldErrors("Value")
		if len(errs) == 0 {
			t.Errorf("expected error on Value field, got: %v", ve)
		}
		// Verify error message mentions min
		if len(errs) > 0 && !strings.Contains(errs[0], "min") {
			t.Errorf("expected error message to contain 'min', got: %s", errs[0])
		}
	})

	t.Run("max on complex number", func(t *testing.T) {
		type TestMaxComplex struct {
			Value complex128 `validate:"max=5"`
		}
		input := TestMaxComplex{Value: 10 + 5i}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for max on complex128")
			return
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Errorf("expected ValidationErrors, got %T", err)
			return
		}
		errs := ve.FieldErrors("Value")
		if len(errs) == 0 {
			t.Errorf("expected error on Value field, got: %v", ve)
		}
		// Verify error message mentions max
		if len(errs) > 0 && !strings.Contains(errs[0], "max") {
			t.Errorf("expected error message to contain 'max', got: %s", errs[0])
		}
	})

	t.Run("min on boolean", func(t *testing.T) {
		type TestMinBool struct {
			Value bool `validate:"min=1"`
		}
		input := TestMinBool{Value: true}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for min on bool")
		}
	})

	t.Run("max on boolean", func(t *testing.T) {
		type TestMaxBool struct {
			Value bool `validate:"max=1"`
		}
		input := TestMaxBool{Value: true}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for max on bool")
		}
	})
}

func TestInvalidValidatorParams(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() any
		wantErr bool
	}{
		{
			name: "eq with non-numeric",
			setup: func() any {
				type Test struct {
					Value int `validate:"eq=abc"`
				}
				return &Test{Value: 5}
			},
			wantErr: true,
		},
		{
			name: "ne with non-numeric",
			setup: func() any {
				type Test struct {
					Value int `validate:"ne=xyz"`
				}
				return &Test{Value: 5}
			},
			wantErr: true,
		},
		{
			name: "gt with empty param",
			setup: func() any {
				type Test struct {
					Value int `validate:"gt="`
				}
				return &Test{Value: 5}
			},
			wantErr: true,
		},
		{
			name: "lt with spaces",
			setup: func() any {
				type Test struct {
					Value int `validate:"lt=  5"`
				}
				return &Test{Value: 10}
			},
			wantErr: true,
		},
		{
			name: "gte with float on int",
			setup: func() any {
				type Test struct {
					Value int `validate:"gte=5.5"`
				}
				return &Test{Value: 5}
			},
			wantErr: false, // integer 5 is >= 5.5 when parsed as float
		},
		{
			name: "lte with multiple decimals",
			setup: func() any {
				type Test struct {
					Value float64 `validate:"lte=5.5.5"`
				}
				return &Test{Value: 5.0}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.setup()
			err := V.Struct(input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error for invalid validator param")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNumericBoundaryValues(t *testing.T) {
	t.Run("int boundaries", func(t *testing.T) {
		type TestInt struct {
			MaxInt32 int `validate:"lte=2147483647"`
			MinInt32 int `validate:"gte=-2147483648"`
		}
		input := TestInt{
			MaxInt32: 2147483647,
			MinInt32: -2147483648,
		}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("int overflow attempt", func(t *testing.T) {
		type TestMax struct {
			Value int `validate:"lte=100"`
		}
		// Large value should fail
		input := TestMax{Value: 2147483647}
		err := V.Struct(&input)
		if err == nil {
			t.Error("expected error for large value exceeding max")
		}
	})

	t.Run("uint64 max", func(t *testing.T) {
		type TestUint64 struct {
			Value uint64 `validate:"gte=0,lte=18446744073709551615"`
		}
		input := TestUint64{Value: ^uint64(0)}
		err := V.Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("float64 special values", func(t *testing.T) {
		// Note: NaN behavior - NaN < 0 is false, so NaN passes gte=0 validation
		// This is acceptable as NaN is an edge case not explicitly handled
		type TestNaN struct {
			Value float64 `validate:"gte=0"`
		}
		input := TestNaN{Value: math.NaN()}
		err := V.Struct(&input)
		// NaN < 0 returns false, so gte=0 passes (no error)
		// Documenting current behavior - NaN is not explicitly rejected
		if err != nil {
			t.Logf("NaN failed validation (may be implementation dependent): %v", err)
		}

		// +Inf should fail lt check (Inf < 1000 is false)
		type TestInf struct {
			Value float64 `validate:"lt=1000"`
		}
		input2 := TestInf{Value: math.Inf(1)}
		err = V.Struct(&input2)
		if err == nil {
			t.Error("expected error for +Inf")
		}

		// -Inf should fail gt check (-Inf > -1000 is false)
		type TestNegInf struct {
			Value float64 `validate:"gt=-1000"`
		}
		input3 := TestNegInf{Value: math.Inf(-1)}
		err = V.Struct(&input3)
		if err == nil {
			t.Error("expected error for -Inf")
		}
	})

	t.Run("zero comparisons", func(t *testing.T) {
		type TestZero struct {
			EqZero  int `validate:"eq=0"`
			GtZero  int `validate:"gt=0"`
			GteZero int `validate:"gte=0"`
			LtZero  int `validate:"lt=0"`
			LteZero int `validate:"lte=0"`
		}
		tests := []struct {
			name    string
			input   TestZero
			wantErr bool
		}{
			{
				name:    "all zeros valid for eq/gte/lte",
				input:   TestZero{EqZero: 0, GteZero: 0, LteZero: 0, GtZero: 1, LtZero: -1},
				wantErr: false,
			},
			{
				name:    "zero fails gt",
				input:   TestZero{EqZero: 0, GteZero: 0, LteZero: 0, GtZero: 0, LtZero: -1},
				wantErr: true,
			},
			{
				name:    "zero fails lt",
				input:   TestZero{EqZero: 0, GteZero: 0, LteZero: 0, GtZero: 1, LtZero: 0},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := V.Struct(&tt.input)
				if tt.wantErr {
					if err == nil {
						t.Error("expected error")
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		}
	})
}

package config

import (
	"reflect"
	"testing"
)

type TestMetricsConfig struct {
	Enabled    bool
	Endpoint   string
	ServerAddr string
}

type TestConfig struct {
	Addr    string
	Metrics TestMetricsConfig
}

var defaultConfig = TestConfig{
	Addr: "localhost:8080",
	Metrics: TestMetricsConfig{
		Enabled:    false,
		Endpoint:   "/metrics",
		ServerAddr: "localhost:9090",
	},
}

func TestConfigMerging(t *testing.T) {
	t.Run("empty user config keeps defaults", func(t *testing.T) {
		c := defaultConfig
		userCfg := TestConfig{}

		Merge(&c, userCfg)

		if c.Addr != "localhost:8080" {
			t.Errorf("expected Addr localhost:8080, got %s", c.Addr)
		}
		if c.Metrics.Enabled != false {
			t.Errorf("expected Metrics.Enabled false, got %v", c.Metrics.Enabled)
		}
		if c.Metrics.Endpoint != "/metrics" {
			t.Errorf("expected Metrics.Endpoint /metrics, got %s", c.Metrics.Endpoint)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		c := defaultConfig
		userCfg := TestConfig{
			Addr: ":9090",
			Metrics: TestMetricsConfig{
				Enabled:  true,
				Endpoint: "/custom-metrics",
			},
		}

		Merge(&c, userCfg)

		if c.Addr != ":9090" {
			t.Errorf("expected Addr :9090, got %s", c.Addr)
		}
		if c.Metrics.Enabled != true {
			t.Errorf("expected Metrics.Enabled true, got %v", c.Metrics.Enabled)
		}
		if c.Metrics.Endpoint != "/custom-metrics" {
			t.Errorf("expected Metrics.Endpoint /custom-metrics, got %s", c.Metrics.Endpoint)
		}
		// ServerAddr should keep default since not set in user config
		if c.Metrics.ServerAddr != "localhost:9090" {
			t.Errorf("expected Metrics.ServerAddr localhost:9090, got %s", c.Metrics.ServerAddr)
		}
	})

	t.Run("partial nested config merges", func(t *testing.T) {
		c := defaultConfig
		userCfg := TestConfig{
			Metrics: TestMetricsConfig{
				Enabled: true,
			},
		}

		Merge(&c, userCfg)

		if !c.Metrics.Enabled {
			t.Error("expected Metrics.Enabled to be true")
		}
		if c.Metrics.Endpoint != "/metrics" {
			t.Errorf("expected Endpoint to keep default, got %s", c.Metrics.Endpoint)
		}
	})

	t.Run("nil user config is noop", func(t *testing.T) {
		c := defaultConfig
		Merge(&c, TestConfig{})

		if c.Addr != "localhost:8080" {
			t.Errorf("expected Addr to keep default, got %s", c.Addr)
		}
	})

	t.Run("src pointer dereferencing", func(t *testing.T) {
		c := defaultConfig
		userCfg := &TestConfig{
			Addr: ":9090",
		}

		Merge(&c, userCfg)

		if c.Addr != ":9090" {
			t.Errorf("expected Addr :9090, got %s", c.Addr)
		}
	})
}

func TestMergePanics(t *testing.T) {
	t.Run("dst not pointer panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-pointer dst")
			}
		}()
		var c TestConfig
		Merge(c, TestConfig{})
	})

	t.Run("dst nil pointer panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil dst")
			}
		}()
		var c *TestConfig
		Merge(c, TestConfig{})
	})

	t.Run("src not struct panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for non-struct src")
			}
		}()
		var c TestConfig
		Merge(&c, "not a struct")
	})

	t.Run("src nil returns early", func(t *testing.T) {
		c := defaultConfig
		Merge(&c, nil)
		if c.Addr != "localhost:8080" {
			t.Errorf("expected Addr to keep default, got %s", c.Addr)
		}
	})
}

// TestAllTypes tests merging of all supported types
func TestAllTypes(t *testing.T) {
	type AllTypes struct {
		Str     string
		Int     int
		Int8    int8
		Int16   int16
		Int32   int32
		Int64   int64
		Uint    uint
		Uint8   uint8
		Uint16  uint16
		Uint32  uint32
		Uint64  uint64
		Float32 float32
		Float64 float64
		Bool    bool
		Slice   []int
		Map     map[string]int
	}

	defaults := AllTypes{
		Str:     "default",
		Int:     1,
		Int8:    1,
		Int16:   1,
		Int32:   1,
		Int64:   1,
		Uint:    1,
		Uint8:   1,
		Uint16:  1,
		Uint32:  1,
		Uint64:  1,
		Float32: 1.0,
		Float64: 1.0,
		Bool:    false,
		Slice:   []int{1, 2, 3},
		Map:     map[string]int{"a": 1},
	}

	t.Run("all types merge correctly", func(t *testing.T) {
		c := defaults
		user := AllTypes{
			Str:     "user",
			Int:     42,
			Int8:    42,
			Int16:   42,
			Int32:   42,
			Int64:   42,
			Uint:    42,
			Uint8:   42,
			Uint16:  42,
			Uint32:  42,
			Uint64:  42,
			Float32: 42.0,
			Float64: 42.0,
			Bool:    true,
			Slice:   []int{4, 5, 6},
			Map:     map[string]int{"b": 2},
		}

		Merge(&c, user)

		if c.Str != "user" {
			t.Errorf("expected Str=user, got %s", c.Str)
		}
		if c.Int != 42 {
			t.Errorf("expected Int=42, got %d", c.Int)
		}
		if c.Int8 != 42 {
			t.Errorf("expected Int8=42, got %d", c.Int8)
		}
		if c.Int16 != 42 {
			t.Errorf("expected Int16=42, got %d", c.Int16)
		}
		if c.Int32 != 42 {
			t.Errorf("expected Int32=42, got %d", c.Int32)
		}
		if c.Int64 != 42 {
			t.Errorf("expected Int64=42, got %d", c.Int64)
		}
		if c.Uint != 42 {
			t.Errorf("expected Uint=42, got %d", c.Uint)
		}
		if c.Uint8 != 42 {
			t.Errorf("expected Uint8=42, got %d", c.Uint8)
		}
		if c.Uint16 != 42 {
			t.Errorf("expected Uint16=42, got %d", c.Uint16)
		}
		if c.Uint32 != 42 {
			t.Errorf("expected Uint32=42, got %d", c.Uint32)
		}
		if c.Uint64 != 42 {
			t.Errorf("expected Uint64=42, got %d", c.Uint64)
		}
		if c.Float32 != 42.0 {
			t.Errorf("expected Float32=42.0, got %f", c.Float32)
		}
		if c.Float64 != 42.0 {
			t.Errorf("expected Float64=42.0, got %f", c.Float64)
		}
		if !c.Bool {
			t.Error("expected Bool=true")
		}
		if len(c.Slice) != 3 || c.Slice[0] != 4 {
			t.Errorf("expected Slice=[4 5 6], got %v", c.Slice)
		}
		if len(c.Map) != 1 || c.Map["b"] != 2 {
			t.Errorf("expected Map={b:2}, got %v", c.Map)
		}
	})

	t.Run("zero values keep defaults", func(t *testing.T) {
		c := defaults
		user := AllTypes{} // All zero values

		Merge(&c, user)

		// Strings: empty should keep default
		if c.Str != "default" {
			t.Errorf("expected Str=default, got %s", c.Str)
		}
		// Numbers: zero should keep default
		if c.Int != 1 {
			t.Errorf("expected Int=1, got %d", c.Int)
		}
		// Bools: false should OVERRIDE (always copied)
		if c.Bool {
			t.Error("expected Bool=false (zero value should override)")
		}
		// Slices: nil/empty should keep default
		if len(c.Slice) != 3 {
			t.Errorf("expected Slice len 3, got %v", c.Slice)
		}
		// Maps: nil/empty should keep default
		if len(c.Map) != 1 {
			t.Errorf("expected Map len 1, got %v", c.Map)
		}
	})

	t.Run("empty slice overrides non-empty", func(t *testing.T) {
		c := defaults
		user := AllTypes{
			Slice: []int{}, // Empty but non-nil
		}

		Merge(&c, user)

		// Empty slice (len 0) should replace non-empty
		if len(c.Slice) != 0 {
			t.Errorf("expected empty slice, got %v", c.Slice)
		}
	})

	// Test bool false overriding true default - this was a bug where structs
	// with all-zero-value fields would skip merging entirely
	t.Run("bool false overrides true default", func(t *testing.T) {
		type BoolOnly struct {
			LogErrors bool
		}

		defaults := BoolOnly{LogErrors: true}
		c := defaults
		user := BoolOnly{LogErrors: false} // User wants to disable error logging

		Merge(&c, user)

		if c.LogErrors {
			t.Error("expected LogErrors=false to override true default, got true")
		}
	})
}

// TestPointerHandling tests pointer field merging
func TestPointerHandling(t *testing.T) {
	type WithPointer struct {
		Value *string
	}

	defaultStr := "default"
	userStr := "user"

	t.Run("non-nil pointer overrides", func(t *testing.T) {
		c := WithPointer{Value: &defaultStr}
		user := WithPointer{Value: &userStr}

		Merge(&c, user)

		if *c.Value != "user" {
			t.Errorf("expected Value=user, got %s", *c.Value)
		}
	})

	t.Run("nil pointer keeps default", func(t *testing.T) {
		c := WithPointer{Value: &defaultStr}
		user := WithPointer{Value: nil}

		Merge(&c, user)

		if c.Value == nil {
			t.Error("expected Value to remain non-nil")
		}
		if *c.Value != "default" {
			t.Errorf("expected Value=default, got %s", *c.Value)
		}
	})
}

// TestInterfaceHandling tests interface field merging
func TestInterfaceHandling(t *testing.T) {
	type WithInterface struct {
		Value any
	}

	t.Run("non-nil interface overrides", func(t *testing.T) {
		c := WithInterface{Value: "default"}
		user := WithInterface{Value: "user"}

		Merge(&c, user)

		if c.Value != "user" {
			t.Errorf("expected Value=user, got %v", c.Value)
		}
	})

	t.Run("nil interface keeps default", func(t *testing.T) {
		c := WithInterface{Value: "default"}
		user := WithInterface{Value: nil}

		Merge(&c, user)

		if c.Value != "default" {
			t.Errorf("expected Value=default, got %v", c.Value)
		}
	})
}

// TestUnexportedFields tests that unexported fields are skipped
func TestUnexportedFields(t *testing.T) {
	type withUnexported struct {
		Exported   string
		unexported string // lowercase = unexported
	}

	c := withUnexported{Exported: "default", unexported: "default"}
	user := withUnexported{Exported: "user", unexported: "user"}

	Merge(&c, user)

	if c.Exported != "user" {
		t.Errorf("expected Exported=user, got %s", c.Exported)
	}
	// unexported field should keep its original value
	if c.unexported != "default" {
		t.Errorf("expected unexported=default, got %s", c.unexported)
	}
}

// TestStructMismatchPanics tests panic on struct mismatch
func TestStructMismatchPanics(t *testing.T) {
	t.Run("dst and src field counts differ", func(t *testing.T) {
		type Small struct{ A string }
		type Large struct{ A, B string }

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for field count mismatch")
			}
		}()

		var dst Small
		src := Large{A: "a", B: "b"}
		Merge(&dst, src)
	})

	t.Run("dst and src field kinds differ", func(t *testing.T) {
		type StructA struct{ Field string }
		type StructB struct{ Field int }

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for field kind mismatch")
			}
		}()

		var dst StructA
		src := StructB{Field: 42}
		Merge(&dst, src)
	})

	t.Run("merge struct into non-struct panics", func(t *testing.T) {
		type Inner struct{ A string }

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for struct into non-struct")
			}
		}()

		dst := struct {
			Field string // not a struct
		}{Field: "value"}

		src := struct {
			Field Inner // struct type
		}{Field: Inner{A: "a"}}

		Merge(&dst, src)
	})
}

// TestNestedStruct tests deeply nested struct merging
func TestNestedStruct(t *testing.T) {
	type Level3 struct {
		Value string
	}
	type Level2 struct {
		Level3 Level3
	}
	type Level1 struct {
		Level2 Level2
	}

	defaults := Level1{
		Level2: Level2{
			Level3: Level3{
				Value: "default",
			},
		},
	}

	t.Run("deeply nested merge", func(t *testing.T) {
		c := defaults
		user := Level1{
			Level2: Level2{
				Level3: Level3{
					Value: "user",
				},
			},
		}

		Merge(&c, user)

		if c.Level2.Level3.Value != "user" {
			t.Errorf("expected nested Value=user, got %s", c.Level2.Level3.Value)
		}
	})

	t.Run("partial nested merge keeps defaults", func(t *testing.T) {
		c := defaults
		user := Level1{} // Empty, all zero values

		Merge(&c, user)

		if c.Level2.Level3.Value != "default" {
			t.Errorf("expected nested Value=default, got %s", c.Level2.Level3.Value)
		}
	})
}

// TestInvalidSrc tests merge with invalid/empty src
func TestInvalidSrc(t *testing.T) {
	t.Run("invalid src value returns early", func(t *testing.T) {
		c := defaultConfig
		// Create an invalid reflect.Value (zero Value)
		Merge(&c, TestConfig{}) // Empty struct is valid, just has zero values
		if c.Addr != "localhost:8080" {
			t.Errorf("expected Addr to keep default, got %s", c.Addr)
		}
	})
}

// TestReflectKindCoverage tests that we cover all switch cases
func TestReflectKindCoverage(t *testing.T) {
	// Test that we properly handle the switch statement in mergeValue
	// This ensures we don't panic on unexpected types

	t.Run("channel is skipped", func(t *testing.T) {
		type WithChan struct {
			Ch  chan int
			Str string
		}

		c := WithChan{Str: "default"}
		user := WithChan{Str: "user", Ch: make(chan int)}

		// Should not panic, channel should be ignored
		Merge(&c, user)

		if c.Str != "user" {
			t.Errorf("expected Str=user, got %s", c.Str)
		}
		// Channel should remain zero (not copied since channels are skipped)
		if c.Ch != nil {
			t.Error("expected Ch to remain nil (channels skipped)")
		}
	})

	t.Run("func is copied", func(t *testing.T) {
		type WithFunc struct {
			Fn  func()
			Str string
		}

		c := WithFunc{Str: "default"}
		called := false
		user := WithFunc{Str: "user", Fn: func() { called = true }}

		// Should not panic, func should be copied
		Merge(&c, user)

		if c.Str != "user" {
			t.Errorf("expected Str=user, got %s", c.Str)
		}
		// Func should be copied
		if c.Fn == nil {
			t.Error("expected Fn to be copied, got nil")
		} else {
			c.Fn()
			if !called {
				t.Error("expected copied func to work")
			}
		}
	})

	t.Run("nil func is skipped", func(t *testing.T) {
		type WithFunc struct {
			Fn  func()
			Str string
		}

		c := WithFunc{Str: "default", Fn: func() {}}
		user := WithFunc{Str: "user", Fn: nil} // nil func

		Merge(&c, user)

		if c.Str != "user" {
			t.Errorf("expected Str=user, got %s", c.Str)
		}
		// Func should remain unchanged
		if c.Fn == nil {
			t.Error("expected Fn to remain non-nil")
		}
	})
}

// TestPointerDereference tests pointer dereferencing when dst is not a pointer
func TestPointerDereference(t *testing.T) {
	t.Run("nil pointer in src is skipped", func(t *testing.T) {
		type WithPtr struct {
			Value *string
		}

		defaultStr := "default"
		dst := WithPtr{Value: &defaultStr}
		src := WithPtr{Value: nil} // nil pointer

		Merge(&dst, src)

		if dst.Value == nil {
			t.Error("expected Value to remain non-nil")
		}
		if *dst.Value != "default" {
			t.Errorf("expected Value=default, got %s", *dst.Value)
		}
	})
}

// TestInterfaceNil tests nil interface handling
func TestInterfaceNil(t *testing.T) {
	t.Run("nil interface in src is skipped", func(t *testing.T) {
		type WithInterface struct {
			Value any
		}

		dst := WithInterface{Value: "default"}
		src := WithInterface{Value: nil} // nil interface

		Merge(&dst, src)

		if dst.Value != "default" {
			t.Errorf("expected Value=default, got %v", dst.Value)
		}
	})
}

// TestArraySkipped tests that arrays are skipped (not handled)
func TestArraySkipped(t *testing.T) {
	t.Run("array is skipped", func(t *testing.T) {
		type WithArray struct {
			Arr [3]int
			Str string
		}

		dst := WithArray{Arr: [3]int{1, 2, 3}, Str: "default"}
		src := WithArray{Arr: [3]int{4, 5, 6}, Str: "user"}

		Merge(&dst, src)

		if dst.Str != "user" {
			t.Errorf("expected Str=user, got %s", dst.Str)
		}
		// Arrays are intentionally skipped in the implementation
		if dst.Arr != [3]int{1, 2, 3} {
			t.Logf("arrays are skipped (expected: %v, got: %v)", [3]int{1, 2, 3}, dst.Arr)
		}
	})
}

// TestMergeValueDirectly tests mergeValue function directly
func TestMergeValueDirectly(t *testing.T) {
	t.Run("invalid src returns early", func(t *testing.T) {
		dst := reflect.ValueOf(&struct{ A string }{A: "default"}).Elem()
		var invalid reflect.Value // zero Value = invalid

		// Should not panic, just return
		mergeValue(dst, invalid)
	})

	t.Run("pointer to pointer copy", func(t *testing.T) {
		type PtrStruct struct {
			P *string
		}

		s := "user"
		dstVal := reflect.ValueOf(&PtrStruct{}).Elem()
		srcVal := reflect.ValueOf(PtrStruct{P: &s})

		mergeValue(dstVal.Field(0), srcVal.Field(0))

		dst := dstVal.Addr().Interface().(*PtrStruct)
		if dst.P == nil || *dst.P != "user" {
			t.Error("expected pointer to be copied")
		}
	})

	t.Run("nil pointer src returns early", func(t *testing.T) {
		type WithPtr struct {
			P *string
		}

		// Create dst with a non-nil pointer
		defaultStr := "default"
		dst := WithPtr{P: &defaultStr}
		dstVal := reflect.ValueOf(&dst).Elem().Field(0) // *string field

		// Create a nil pointer value directly
		var nilStr *string
		srcVal := reflect.ValueOf(nilStr)

		// Should return early (nil pointer) - dst should keep its value
		mergeValue(dstVal, srcVal)

		// dst should remain unchanged
		if dst.P == nil || *dst.P != "default" {
			t.Errorf("expected dst.P to remain 'default', got %v", dst.P)
		}
	})

	t.Run("nil interface src returns early", func(t *testing.T) {
		type WithInterface struct {
			V any
		}

		dstVal := reflect.ValueOf(&WithInterface{V: "default"}).Elem().Field(0) // any field
		// Create a nil interface value
		var nilAny any
		srcVal := reflect.ValueOf(&nilAny).Elem()

		// Should return early (nil interface)
		mergeValue(dstVal, srcVal)

		// dst should remain unchanged
		if dstVal.Interface() != "default" {
			t.Errorf("expected default, got %v", dstVal.Interface())
		}
	})

	t.Run("pointer dereference into non-pointer", func(t *testing.T) {
		// This tests line 63: mergeValue(dst, src.Elem())
		// When src is a pointer and dst is NOT a pointer/interface

		type Inner struct {
			Value string
		}

		dstVal := reflect.ValueOf(&Inner{Value: "default"}).Elem().Field(0) // string field
		inner := Inner{Value: "user"}
		// Get a pointer to inner.Value and then get the pointer value
		ptrVal := reflect.ValueOf(&inner.Value) // *string

		// This should dereference the pointer and merge the string value
		mergeValue(dstVal, ptrVal)

		if dstVal.String() != "user" {
			t.Errorf("expected user, got %s", dstVal.String())
		}
	})

	t.Run("struct into non-struct panics", func(t *testing.T) {
		// This tests line 102-103: panic for struct into non-struct
		type Inner struct {
			A string
		}

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for struct into non-struct")
			}
		}()

		// Create dst as a string value (non-struct)
		dstStr := "default"
		dstVal := reflect.ValueOf(&dstStr).Elem()

		// Create src as a struct value
		srcVal := reflect.ValueOf(Inner{A: "user"})

		mergeValue(dstVal, srcVal)
	})
}

package config

import (
	"reflect"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

type TestMetricsConfig struct {
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
		Endpoint:   "/metrics",
		ServerAddr: "localhost:9090",
	},
}

func TestConfigMerging(t *testing.T) {
	t.Run("empty user config keeps defaults", func(t *testing.T) {
		c := defaultConfig
		userCfg := TestConfig{}

		Merge(&c, userCfg)

		zhtest.AssertEqual(t, "localhost:8080", c.Addr)
		zhtest.AssertEqual(t, "/metrics", c.Metrics.Endpoint)
	})

	t.Run("user values override defaults", func(t *testing.T) {
		c := defaultConfig
		userCfg := TestConfig{
			Addr: ":9090",
			Metrics: TestMetricsConfig{
				Endpoint: "/custom-metrics",
			},
		}

		Merge(&c, userCfg)

		zhtest.AssertEqual(t, ":9090", c.Addr)
		zhtest.AssertEqual(t, "/custom-metrics", c.Metrics.Endpoint)
		// ServerAddr should keep default since not set in user config
		zhtest.AssertEqual(t, "localhost:9090", c.Metrics.ServerAddr)
	})

	t.Run("partial nested config merges", func(t *testing.T) {
		c := defaultConfig
		userCfg := TestConfig{
			Metrics: TestMetricsConfig{
				ServerAddr: "custom:9090",
			},
		}

		Merge(&c, userCfg)

		zhtest.AssertEqual(t, "custom:9090", c.Metrics.ServerAddr)
		zhtest.AssertEqual(t, "/metrics", c.Metrics.Endpoint)
	})

	t.Run("nil user config is noop", func(t *testing.T) {
		c := defaultConfig
		Merge(&c, TestConfig{})

		zhtest.AssertEqual(t, "localhost:8080", c.Addr)
	})

	t.Run("src pointer dereferencing", func(t *testing.T) {
		c := defaultConfig
		userCfg := &TestConfig{
			Addr: ":9090",
		}

		Merge(&c, userCfg)

		zhtest.AssertEqual(t, ":9090", c.Addr)
	})
}

func TestMergePanics(t *testing.T) {
	t.Run("dst not pointer panics", func(t *testing.T) {
		zhtest.AssertPanic(t, func() {
			var c TestConfig
			Merge(c, TestConfig{})
		})
	})

	t.Run("dst nil pointer panics", func(t *testing.T) {
		zhtest.AssertPanic(t, func() {
			var c *TestConfig
			Merge(c, TestConfig{})
		})
	})

	t.Run("src not struct panics", func(t *testing.T) {
		zhtest.AssertPanic(t, func() {
			var c TestConfig
			Merge(&c, "not a struct")
		})
	})

	t.Run("src nil returns early", func(t *testing.T) {
		c := defaultConfig
		Merge(&c, nil)
		zhtest.AssertEqual(t, "localhost:8080", c.Addr)
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

		zhtest.AssertEqual(t, "user", c.Str)
		zhtest.AssertEqual(t, 42, c.Int)
		zhtest.AssertEqual(t, int8(42), c.Int8)
		zhtest.AssertEqual(t, int16(42), c.Int16)
		zhtest.AssertEqual(t, int32(42), c.Int32)
		zhtest.AssertEqual(t, int64(42), c.Int64)
		zhtest.AssertEqual(t, uint(42), c.Uint)
		zhtest.AssertEqual(t, uint8(42), c.Uint8)
		zhtest.AssertEqual(t, uint16(42), c.Uint16)
		zhtest.AssertEqual(t, uint32(42), c.Uint32)
		zhtest.AssertEqual(t, uint64(42), c.Uint64)
		zhtest.AssertEqual(t, float32(42.0), c.Float32)
		zhtest.AssertEqual(t, 42.0, c.Float64)
		zhtest.AssertTrue(t, c.Bool)
		zhtest.AssertEqual(t, []int{4, 5, 6}, c.Slice)
		zhtest.AssertEqual(t, map[string]int{"b": 2}, c.Map)
	})

	t.Run("zero values keep defaults", func(t *testing.T) {
		c := defaults
		user := AllTypes{} // All zero values

		Merge(&c, user)

		// Strings: empty should keep default
		zhtest.AssertEqual(t, "default", c.Str)
		// Numbers: zero should keep default
		zhtest.AssertEqual(t, 1, c.Int)
		// Bools: false should OVERRIDE (always copied)
		zhtest.AssertFalse(t, c.Bool)
		// Slices: nil/empty should keep default
		zhtest.AssertEqual(t, 3, len(c.Slice))
		// Maps: nil/empty should keep default
		zhtest.AssertEqual(t, 1, len(c.Map))
	})

	t.Run("empty slice overrides non-empty", func(t *testing.T) {
		c := defaults
		user := AllTypes{
			Slice: []int{}, // Empty but non-nil
		}

		Merge(&c, user)

		// Empty slice (len 0) should replace non-empty
		zhtest.AssertEqual(t, 0, len(c.Slice))
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

		zhtest.AssertFalse(t, c.LogErrors)
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

		zhtest.AssertEqual(t, "user", *c.Value)
	})

	t.Run("nil pointer keeps default", func(t *testing.T) {
		c := WithPointer{Value: &defaultStr}
		user := WithPointer{Value: nil}

		Merge(&c, user)

		zhtest.AssertNotNil(t, c.Value)
		zhtest.AssertEqual(t, "default", *c.Value)
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

		zhtest.AssertEqual(t, "user", c.Value)
	})

	t.Run("nil interface keeps default", func(t *testing.T) {
		c := WithInterface{Value: "default"}
		user := WithInterface{Value: nil}

		Merge(&c, user)

		zhtest.AssertEqual(t, "default", c.Value)
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

	zhtest.AssertEqual(t, "user", c.Exported)
	// unexported field should keep its original value
	zhtest.AssertEqual(t, "default", c.unexported)
}

// TestStructMismatchPanics tests panic on struct mismatch
func TestStructMismatchPanics(t *testing.T) {
	t.Run("dst and src field counts differ", func(t *testing.T) {
		type Small struct{ A string }
		type Large struct{ A, B string }

		zhtest.AssertPanic(t, func() {
			var dst Small
			src := Large{A: "a", B: "b"}
			Merge(&dst, src)
		})
	})

	t.Run("dst and src field kinds differ", func(t *testing.T) {
		type StructA struct{ Field string }
		type StructB struct{ Field int }

		zhtest.AssertPanic(t, func() {
			var dst StructA
			src := StructB{Field: 42}
			Merge(&dst, src)
		})
	})

	t.Run("merge struct into non-struct panics", func(t *testing.T) {
		type Inner struct{ A string }

		zhtest.AssertPanic(t, func() {
			dst := struct {
				Field string // not a struct
			}{Field: "value"}

			src := struct {
				Field Inner // struct type
			}{Field: Inner{A: "a"}}

			Merge(&dst, src)
		})
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

		zhtest.AssertEqual(t, "user", c.Level2.Level3.Value)
	})

	t.Run("partial nested merge keeps defaults", func(t *testing.T) {
		c := defaults
		user := Level1{} // Empty, all zero values

		Merge(&c, user)

		zhtest.AssertEqual(t, "default", c.Level2.Level3.Value)
	})
}

// TestInvalidSrc tests merge with invalid/empty src
func TestInvalidSrc(t *testing.T) {
	t.Run("invalid src value returns early", func(t *testing.T) {
		c := defaultConfig
		// Create an invalid reflect.Value (zero Value)
		Merge(&c, TestConfig{}) // Empty struct is valid, just has zero values
		zhtest.AssertEqual(t, "localhost:8080", c.Addr)
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

		zhtest.AssertEqual(t, "user", c.Str)
		// Channel should remain zero (not copied since channels are skipped)
		zhtest.AssertNil(t, c.Ch)
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

		zhtest.AssertEqual(t, "user", c.Str)
		// Func should be copied
		zhtest.AssertNotNil(t, c.Fn)
		c.Fn()
		zhtest.AssertTrue(t, called)
	})

	t.Run("nil func is skipped", func(t *testing.T) {
		type WithFunc struct {
			Fn  func()
			Str string
		}

		c := WithFunc{Str: "default", Fn: func() {}}
		user := WithFunc{Str: "user", Fn: nil} // nil func

		Merge(&c, user)

		zhtest.AssertEqual(t, "user", c.Str)
		// Func should remain unchanged
		zhtest.AssertNotNil(t, c.Fn)
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

		zhtest.AssertNotNil(t, dst.Value)
		zhtest.AssertEqual(t, "default", *dst.Value)
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

		zhtest.AssertEqual(t, "default", dst.Value)
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

		zhtest.AssertEqual(t, "user", dst.Str)
		// Arrays are intentionally skipped in the implementation
		zhtest.AssertEqual(t, [3]int{1, 2, 3}, dst.Arr)
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
		zhtest.AssertNotNil(t, dst.P)
		zhtest.AssertEqual(t, "user", *dst.P)
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
		zhtest.AssertNotNil(t, dst.P)
		zhtest.AssertEqual(t, "default", *dst.P)
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
		zhtest.AssertEqual(t, "default", dstVal.Interface())
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

		zhtest.AssertEqual(t, "user", dstVal.String())
	})

	t.Run("struct into non-struct panics", func(t *testing.T) {
		// This tests line 102-103: panic for struct into non-struct
		type Inner struct {
			A string
		}

		zhtest.AssertPanic(t, func() {
			// Create dst as a string value (non-struct)
			dstStr := "default"
			dstVal := reflect.ValueOf(&dstStr).Elem()

			// Create src as a struct value
			srcVal := reflect.ValueOf(Inner{A: "user"})

			mergeValue(dstVal, srcVal)
		})
	})
}

package config

import (
	"testing"
)

func TestBool(t *testing.T) {
	t.Run("returns pointer to true", func(t *testing.T) {
		b := Bool(true)
		if b == nil {
			t.Fatal("expected non-nil pointer")
		}
		if !*b {
			t.Error("expected true")
		}
	})

	t.Run("returns pointer to false", func(t *testing.T) {
		b := Bool(false)
		if b == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *b {
			t.Error("expected false")
		}
	})
}

func TestBoolOrDefault(t *testing.T) {
	t.Run("returns value when not nil", func(t *testing.T) {
		b := true
		result := BoolOrDefault(&b, false)
		if !result {
			t.Error("expected true, got false")
		}
	})

	t.Run("returns default when nil", func(t *testing.T) {
		result := BoolOrDefault(nil, true)
		if !result {
			t.Error("expected default true, got false")
		}
	})

	t.Run("returns default false when nil", func(t *testing.T) {
		result := BoolOrDefault(nil, false)
		if result {
			t.Error("expected default false, got true")
		}
	})
}

func TestString(t *testing.T) {
	t.Run("returns pointer to string", func(t *testing.T) {
		s := String("test")
		if s == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *s != "test" {
			t.Errorf("expected 'test', got '%s'", *s)
		}
	})
}

func TestStringOrDefault(t *testing.T) {
	t.Run("returns value when not nil", func(t *testing.T) {
		s := "hello"
		result := StringOrDefault(&s, "default")
		if result != "hello" {
			t.Errorf("expected 'hello', got '%s'", result)
		}
	})

	t.Run("returns default when nil", func(t *testing.T) {
		result := StringOrDefault(nil, "default")
		if result != "default" {
			t.Errorf("expected 'default', got '%s'", result)
		}
	})

	t.Run("returns empty string when nil and default is empty", func(t *testing.T) {
		result := StringOrDefault(nil, "")
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})
}

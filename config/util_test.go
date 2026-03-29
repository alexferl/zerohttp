package config

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestBool(t *testing.T) {
	t.Run("returns pointer to true", func(t *testing.T) {
		b := Bool(true)
		zhtest.AssertNotNil(t, b)
		zhtest.AssertTrue(t, *b)
	})

	t.Run("returns pointer to false", func(t *testing.T) {
		b := Bool(false)
		zhtest.AssertNotNil(t, b)
		zhtest.AssertFalse(t, *b)
	})
}

func TestBoolOrDefault(t *testing.T) {
	t.Run("returns value when not nil", func(t *testing.T) {
		b := true
		result := BoolOrDefault(&b, false)
		zhtest.AssertTrue(t, result)
	})

	t.Run("returns default when nil", func(t *testing.T) {
		result := BoolOrDefault(nil, true)
		zhtest.AssertTrue(t, result)
	})

	t.Run("returns default false when nil", func(t *testing.T) {
		result := BoolOrDefault(nil, false)
		zhtest.AssertFalse(t, result)
	})
}

func TestString(t *testing.T) {
	t.Run("returns pointer to string", func(t *testing.T) {
		s := String("test")
		zhtest.AssertNotNil(t, s)
		zhtest.AssertEqual(t, "test", *s)
	})
}

func TestStringOrDefault(t *testing.T) {
	t.Run("returns value when not nil", func(t *testing.T) {
		s := "hello"
		result := StringOrDefault(&s, "default")
		zhtest.AssertEqual(t, "hello", result)
	})

	t.Run("returns default when nil", func(t *testing.T) {
		result := StringOrDefault(nil, "default")
		zhtest.AssertEqual(t, "default", result)
	})

	t.Run("returns empty string when nil and default is empty", func(t *testing.T) {
		result := StringOrDefault(nil, "")
		zhtest.AssertEmpty(t, result)
	})
}

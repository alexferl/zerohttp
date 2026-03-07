package config

import (
	"reflect"
	"testing"
)

func TestSetHeaderConfig_DefaultValues(t *testing.T) {
	cfg := DefaultSetHeaderConfig
	if cfg.Headers == nil {
		t.Error("expected default headers to be initialized, not nil")
	}
	if len(cfg.Headers) != 0 {
		t.Errorf("expected default headers to be empty, got %d headers", len(cfg.Headers))
	}
}

func TestWithSetHeadersOption(t *testing.T) {
	t.Run("multiple headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Custom-Header":  "custom-value",
			"X-API-Version":    "v1.0",
			"X-Request-Source": "api-gateway",
		}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if len(cfg.Headers) != 3 {
			t.Errorf("expected 3 headers, got %d", len(cfg.Headers))
		}
		if !reflect.DeepEqual(cfg.Headers, headers) {
			t.Errorf("expected headers = %v, got %v", headers, cfg.Headers)
		}

		// Test individual header values
		if cfg.Headers["X-Custom-Header"] != "custom-value" {
			t.Errorf("expected X-Custom-Header = 'custom-value', got %s", cfg.Headers["X-Custom-Header"])
		}
		if cfg.Headers["X-API-Version"] != "v1.0" {
			t.Errorf("expected X-API-Version = 'v1.0', got %s", cfg.Headers["X-API-Version"])
		}
		if cfg.Headers["X-Request-Source"] != "api-gateway" {
			t.Errorf("expected X-Request-Source = 'api-gateway', got %s", cfg.Headers["X-Request-Source"])
		}
	})

	t.Run("single header", func(t *testing.T) {
		headers := map[string]string{"X-Single-Header": "single-value"}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if len(cfg.Headers) != 1 {
			t.Errorf("expected 1 header, got %d", len(cfg.Headers))
		}
		if cfg.Headers["X-Single-Header"] != "single-value" {
			t.Errorf("expected X-Single-Header = 'single-value', got %s", cfg.Headers["X-Single-Header"])
		}
	})

	t.Run("overwrite headers", func(t *testing.T) {
		firstHeaders := map[string]string{
			"X-Test-Header": "first-value",
			"X-Other":       "other-value",
		}
		secondHeaders := map[string]string{
			"X-Test-Header": "second-value",
			"X-New-Header":  "new-value",
		}

		cfg := DefaultSetHeaderConfig
		WithSetHeaders(firstHeaders)(&cfg)
		WithSetHeaders(secondHeaders)(&cfg)

		// Should only have the second set of headers
		if len(cfg.Headers) != 2 {
			t.Errorf("expected 2 headers after overwrite, got %d", len(cfg.Headers))
		}
		if cfg.Headers["X-Test-Header"] != "second-value" {
			t.Errorf("expected X-Test-Header = 'second-value', got %s", cfg.Headers["X-Test-Header"])
		}
		if cfg.Headers["X-New-Header"] != "new-value" {
			t.Errorf("expected X-New-Header = 'new-value', got %s", cfg.Headers["X-New-Header"])
		}
		// First header should be gone
		if _, exists := cfg.Headers["X-Other"]; exists {
			t.Error("expected X-Other header to be overwritten and not exist")
		}
	})
}

func TestSetHeaderConfig_EdgeCases(t *testing.T) {
	t.Run("empty headers", func(t *testing.T) {
		emptyHeaders := map[string]string{}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(emptyHeaders)(&cfg)

		if cfg.Headers == nil {
			t.Error("expected headers map to be initialized, not nil")
		}
		if len(cfg.Headers) != 0 {
			t.Errorf("expected empty headers map, got %d entries", len(cfg.Headers))
		}
	})

	t.Run("nil headers", func(t *testing.T) {
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(nil)(&cfg)
		if cfg.Headers != nil {
			t.Error("expected headers to remain nil when nil is passed")
		}
	})

	t.Run("case sensitive keys", func(t *testing.T) {
		headers := map[string]string{
			"x-custom-header": "lowercase",
			"X-Custom-Header": "titlecase",
			"X-CUSTOM-HEADER": "uppercase",
		}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if len(cfg.Headers) != 3 {
			t.Errorf("expected 3 headers (case-sensitive keys), got %d", len(cfg.Headers))
		}
		// Test that all case variations are preserved
		if cfg.Headers["x-custom-header"] != "lowercase" {
			t.Error("expected lowercase header key to be preserved")
		}
		if cfg.Headers["X-Custom-Header"] != "titlecase" {
			t.Error("expected title-case header key to be preserved")
		}
		if cfg.Headers["X-CUSTOM-HEADER"] != "uppercase" {
			t.Error("expected uppercase header key to be preserved")
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		headers := map[string]string{
			"X-Empty-Header":  "",
			"X-Normal-Header": "value",
			"X-Another-Empty": "",
		}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if len(cfg.Headers) != 3 {
			t.Errorf("expected 3 headers, got %d", len(cfg.Headers))
		}
		if cfg.Headers["X-Empty-Header"] != "" {
			t.Errorf("expected X-Empty-Header = '', got %s", cfg.Headers["X-Empty-Header"])
		}
		if cfg.Headers["X-Normal-Header"] != "value" {
			t.Errorf("expected X-Normal-Header = 'value', got %s", cfg.Headers["X-Normal-Header"])
		}
		if cfg.Headers["X-Another-Empty"] != "" {
			t.Errorf("expected X-Another-Empty = '', got %s", cfg.Headers["X-Another-Empty"])
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := SetHeaderConfig{} // Zero values
		if cfg.Headers != nil {
			t.Errorf("expected zero headers = nil, got %v", cfg.Headers)
		}
	})
}

func TestSetHeaderConfig_SpecialValues(t *testing.T) {
	t.Run("special character values", func(t *testing.T) {
		headers := map[string]string{
			"X-JSON-Data":     `{"key":"value","number":123}`,
			"X-URL-Data":      "https://example.com/path?param=value&other=123",
			"X-Special-Chars": "value with spaces, commas; semicolons: colons",
			"X-Unicode":       "héllo wørld ñoñó",
			"X-Quotes":        `"quoted value" and 'single quotes'`,
		}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if len(cfg.Headers) != 5 {
			t.Errorf("expected 5 headers, got %d", len(cfg.Headers))
		}
		for key, expectedValue := range headers {
			if cfg.Headers[key] != expectedValue {
				t.Errorf("expected header %s = %s, got %s", key, expectedValue, cfg.Headers[key])
			}
		}
	})

	t.Run("long values", func(t *testing.T) {
		longValue := "This is a very long header value that might be used for embedding large amounts of data or configuration information directly in HTTP headers, which while not recommended for extremely large data, is sometimes necessary for certain use cases."
		headers := map[string]string{
			"X-Long-Header": longValue,
			"X-Short":       "short",
		}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if cfg.Headers["X-Long-Header"] != longValue {
			t.Error("expected long header value to be preserved exactly")
		}
		if cfg.Headers["X-Short"] != "short" {
			t.Error("expected short header value to be preserved")
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		headers := map[string]string{
			"X-Leading-Space":   " value-with-leading-space",
			"X-Trailing-Space":  "value-with-trailing-space ",
			"X-Both-Spaces":     " value with both spaces ",
			"X-Internal-Spaces": "value with internal spaces",
			"X-Tab-Character":   "value\twith\ttabs",
			"X-Newline":         "value\nwith\nnewlines",
		}
		cfg := DefaultSetHeaderConfig
		WithSetHeaders(headers)(&cfg)

		if len(cfg.Headers) != 6 {
			t.Errorf("expected 6 whitespace headers, got %d", len(cfg.Headers))
		}
		// Test that whitespace is preserved exactly as provided
		for key, expectedValue := range headers {
			if cfg.Headers[key] != expectedValue {
				t.Errorf("expected whitespace header %s = %q, got %q", key, expectedValue, cfg.Headers[key])
			}
		}
	})
}

package setheader

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestSetHeaderConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertNotNil(t, cfg.Headers)
	zhtest.AssertEqual(t, 0, len(cfg.Headers))
}

func TestSetHeaderConfig_StructAssignment(t *testing.T) {
	t.Run("multiple headers", func(t *testing.T) {
		headers := map[string]string{
			"X-Custom-Header":  "custom-value",
			"X-API-Version":    "v1.0",
			"X-Request-Source": "api-gateway",
		}
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, 3, len(cfg.Headers))
		zhtest.AssertDeepEqual(t, headers, cfg.Headers)

		// Test individual header values
		zhtest.AssertEqual(t, "custom-value", cfg.Headers["X-Custom-Header"])
		zhtest.AssertEqual(t, "v1.0", cfg.Headers["X-API-Version"])
		zhtest.AssertEqual(t, "api-gateway", cfg.Headers["X-Request-Source"])
	})

	t.Run("single header", func(t *testing.T) {
		headers := map[string]string{"X-Single-Header": "single-value"}
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, 1, len(cfg.Headers))
		zhtest.AssertEqual(t, "single-value", cfg.Headers["X-Single-Header"])
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

		cfg := Config{
			Headers: firstHeaders,
		}
		// Reassign to simulate overwrite
		cfg.Headers = secondHeaders

		// Should only have the second set of headers
		zhtest.AssertEqual(t, 2, len(cfg.Headers))
		zhtest.AssertEqual(t, "second-value", cfg.Headers["X-Test-Header"])
		zhtest.AssertEqual(t, "new-value", cfg.Headers["X-New-Header"])
		// First header should be gone
		_, exists := cfg.Headers["X-Other"]
		zhtest.AssertFalse(t, exists)
	})
}

func TestSetHeaderConfig_EdgeCases(t *testing.T) {
	t.Run("empty headers", func(t *testing.T) {
		emptyHeaders := map[string]string{}
		cfg := Config{
			Headers: emptyHeaders,
		}

		zhtest.AssertNotNil(t, cfg.Headers)
		zhtest.AssertEqual(t, 0, len(cfg.Headers))
	})

	t.Run("nil headers", func(t *testing.T) {
		cfg := Config{
			Headers: nil,
		}
		zhtest.AssertNil(t, cfg.Headers)
	})

	t.Run("case sensitive keys", func(t *testing.T) {
		headers := map[string]string{
			"x-custom-header": "lowercase",
			"X-Custom-Header": "titlecase",
			"X-CUSTOM-HEADER": "uppercase",
		}
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, 3, len(cfg.Headers))
		// Test that all case variations are preserved
		zhtest.AssertEqual(t, "lowercase", cfg.Headers["x-custom-header"])
		zhtest.AssertEqual(t, "titlecase", cfg.Headers["X-Custom-Header"])
		zhtest.AssertEqual(t, "uppercase", cfg.Headers["X-CUSTOM-HEADER"])
	})

	t.Run("empty string values", func(t *testing.T) {
		headers := map[string]string{
			"X-Empty-Header":  "",
			"X-Normal-Header": "value",
			"X-Another-Empty": "",
		}
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, 3, len(cfg.Headers))
		zhtest.AssertEqual(t, "", cfg.Headers["X-Empty-Header"])
		zhtest.AssertEqual(t, "value", cfg.Headers["X-Normal-Header"])
		zhtest.AssertEqual(t, "", cfg.Headers["X-Another-Empty"])
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertNil(t, cfg.Headers)
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
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, 5, len(cfg.Headers))
		for key, expectedValue := range headers {
			zhtest.AssertEqual(t, expectedValue, cfg.Headers[key])
		}
	})

	t.Run("long values", func(t *testing.T) {
		longValue := "This is a very long header value that might be used for embedding large amounts of data or configuration information directly in HTTP headers, which while not recommended for extremely large data, is sometimes necessary for certain use cases."
		headers := map[string]string{
			"X-Long-Header": longValue,
			"X-Short":       "short",
		}
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, longValue, cfg.Headers["X-Long-Header"])
		zhtest.AssertEqual(t, "short", cfg.Headers["X-Short"])
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
		cfg := Config{
			Headers: headers,
		}

		zhtest.AssertEqual(t, 6, len(cfg.Headers))
		// Test that whitespace is preserved exactly as provided
		for key, expectedValue := range headers {
			zhtest.AssertEqual(t, expectedValue, cfg.Headers[key])
		}
	})
}

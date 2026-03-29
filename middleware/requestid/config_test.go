package requestid

import (
	"regexp"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequestIDConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, "X-Request-Id", cfg.Header)
	zhtest.AssertNotNil(t, cfg.Generator)
	zhtest.AssertEqual(t, ContextKey, cfg.ContextKey)

	// Test context key type
	_, ok := cfg.ContextKey.(contextKey)
	zhtest.AssertTrue(t, ok)
}

func TestGenerateRequestID(t *testing.T) {
	t.Run("format validation", func(t *testing.T) {
		hexPattern := regexp.MustCompile("^[a-f0-9]+$")
		for range 10 {
			id := GenerateRequestID()
			zhtest.AssertEqual(t, 32, len(id))
			zhtest.AssertTrue(t, hexPattern.MatchString(id))
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		ids := make(map[string]bool)
		iterations := 100
		for range iterations {
			id := GenerateRequestID()
			zhtest.AssertFalse(t, ids[id])
			ids[id] = true
		}
		zhtest.AssertEqual(t, iterations, len(ids))
	})

	t.Run("performance", func(t *testing.T) {
		start := time.Now()
		iterations := 1000
		for range iterations {
			id := GenerateRequestID()
			zhtest.AssertEqual(t, 32, len(id))
		}
		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(iterations)
		if avgTime > time.Millisecond {
			t.Logf("Generator average time: %v (might be slow but not necessarily a failure)", avgTime)
		}
	})

	t.Run("concurrent generation", func(t *testing.T) {
		const numGoroutines = 10
		const numIterations = 10
		ids := make(chan string, numGoroutines*numIterations)

		for range numGoroutines {
			go func() {
				for j := 0; j < numIterations; j++ {
					ids <- GenerateRequestID()
				}
			}()
		}

		uniqueIDs := make(map[string]bool)
		for range numGoroutines * numIterations {
			id := <-ids
			zhtest.AssertFalse(t, uniqueIDs[id])
			uniqueIDs[id] = true
		}

		expectedCount := numGoroutines * numIterations
		zhtest.AssertEqual(t, expectedCount, len(uniqueIDs))
	})
}

func TestRequestIDConfig_StructAssignment(t *testing.T) {
	t.Run("header assignment", func(t *testing.T) {
		cfg := Config{
			Header:     "X-Trace-Id",
			Generator:  GenerateRequestID,
			ContextKey: ContextKey,
		}
		zhtest.AssertEqual(t, "X-Trace-Id", cfg.Header)
	})

	t.Run("generator assignment", func(t *testing.T) {
		customGenerator := func() string { return "custom-id-123" }
		cfg := Config{
			Header:     "X-Request-Id",
			Generator:  customGenerator,
			ContextKey: ContextKey,
		}
		zhtest.AssertNotNil(t, cfg.Generator)
		zhtest.AssertEqual(t, "custom-id-123", cfg.Generator())
	})

	t.Run("context key assignment", func(t *testing.T) {
		// Custom context key using a different struct type
		type customKey struct{}
		customKeyInstance := customKey{}
		cfg := Config{
			Header:     "X-Request-Id",
			Generator:  GenerateRequestID,
			ContextKey: customKeyInstance,
		}
		zhtest.AssertEqual(t, customKeyInstance, cfg.ContextKey)
	})

	t.Run("multiple fields assignment", func(t *testing.T) {
		customGenerator := func() string { return "multi-option-id" }
		cfg := Config{
			Header:     "X-Custom-Request-Id",
			Generator:  customGenerator,
			ContextKey: ContextKey,
		}

		zhtest.AssertEqual(t, "X-Custom-Request-Id", cfg.Header)
		zhtest.AssertNotNil(t, cfg.Generator)
		zhtest.AssertEqual(t, "multi-option-id", cfg.Generator())
		zhtest.AssertEqual(t, ContextKey, cfg.ContextKey)
	})
}

func TestRequestIDConfig_CommonScenarios(t *testing.T) {
	t.Run("common headers", func(t *testing.T) {
		headers := []string{"X-Request-Id", "X-Trace-Id", "X-Correlation-Id", "Request-ID", "Trace-ID", "X-Request-UUID", "X-Session-Id"}
		for _, header := range headers {
			t.Run(header, func(t *testing.T) {
				cfg := Config{
					Header:     header,
					Generator:  GenerateRequestID,
					ContextKey: ContextKey,
				}
				zhtest.AssertEqual(t, header, cfg.Header)
			})
		}
	})

	t.Run("custom generators", func(t *testing.T) {
		tests := []struct {
			name      string
			generator func() string
			expected  string
		}{
			{"UUID-style", func() string { return "123e4567-e89b-12d3-a456-426614174000" }, "123e4567-e89b-12d3-a456-426614174000"},
			{"timestamp-based", func() string { return "req-12345" }, "req-12345"},
			{"counter-based", func() string { return "req-001" }, "req-001"},
			{"short hex", func() string { return "abc123" }, "abc123"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := Config{
					Header:     "X-Request-Id",
					Generator:  tt.generator,
					ContextKey: ContextKey,
				}
				zhtest.AssertEqual(t, tt.expected, cfg.Generator())
			})
		}
	})

	t.Run("custom context keys", func(t *testing.T) {
		tests := []struct {
			name string
			key  any
		}{
			{"default struct key", ContextKey},
			{"custom string key", "my_request_id"},
			{"custom int key", 42},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := Config{
					Header:     "X-Request-Id",
					Generator:  GenerateRequestID,
					ContextKey: tt.key,
				}
				zhtest.AssertEqual(t, tt.key, cfg.ContextKey)
			})
		}
	})

	t.Run("special character headers", func(t *testing.T) {
		headers := []string{
			"X-Request-Id", "x-request-id", "REQUEST-ID", "X_Request_ID",
			"X-Request-Id-123", "Custom.Request.ID", "Request-Id-With-Dashes",
			"X-Request-Id_With_Underscores",
		}

		for _, header := range headers {
			t.Run(header, func(t *testing.T) {
				cfg := Config{
					Header:     header,
					Generator:  GenerateRequestID,
					ContextKey: ContextKey,
				}
				zhtest.AssertEqual(t, header, cfg.Header)
			})
		}
	})
}

func TestRequestIDConfig_EdgeCases(t *testing.T) {
	t.Run("nil generator", func(t *testing.T) {
		cfg := Config{
			Header:     "X-Request-Id",
			Generator:  nil,
			ContextKey: ContextKey,
		}
		zhtest.AssertNil(t, cfg.Generator)
	})

	t.Run("empty string context key", func(t *testing.T) {
		cfg := Config{
			Header:     "",
			Generator:  GenerateRequestID,
			ContextKey: "",
		}
		zhtest.AssertEmpty(t, cfg.Header)
		zhtest.AssertEmpty(t, cfg.ContextKey)
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertEmpty(t, cfg.Header)
		zhtest.AssertNil(t, cfg.Generator)
		zhtest.AssertNil(t, cfg.ContextKey)
	})
}

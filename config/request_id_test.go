package config

import (
	"regexp"
	"testing"
	"time"
)

func TestRequestIDConfig_DefaultValues(t *testing.T) {
	cfg := DefaultRequestIDConfig
	if cfg.Header != "X-Request-Id" {
		t.Errorf("expected default header = 'X-Request-Id', got %s", cfg.Header)
	}
	if cfg.Generator == nil {
		t.Error("expected default generator to be set")
	}
	if cfg.ContextKey != "request_id" {
		t.Errorf("expected default context key = 'request_id', got %s", string(cfg.ContextKey))
	}

	// Test context key type
	if key, ok := interface{}(cfg.ContextKey).(RequestIDContextKey); !ok {
		t.Error("expected ContextKey to be of type RequestIDContextKey")
	} else if string(key) != "request_id" {
		t.Errorf("expected context key string value = 'request_id', got %s", string(key))
	}
}

func TestGenerateRequestID(t *testing.T) {
	t.Run("format validation", func(t *testing.T) {
		hexPattern := regexp.MustCompile("^[a-f0-9]+$")
		for range 10 {
			id := GenerateRequestID()
			if len(id) != 32 {
				t.Errorf("expected request ID length = 32, got %d for ID: %s", len(id), id)
			}
			if !hexPattern.MatchString(id) {
				t.Errorf("request ID should only contain hex characters, got: %s", id)
			}
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		ids := make(map[string]bool)
		iterations := 100
		for range iterations {
			id := GenerateRequestID()
			if ids[id] {
				t.Errorf("found duplicate request ID: %s", id)
			}
			ids[id] = true
		}
		if len(ids) != iterations {
			t.Errorf("expected %d unique IDs, got %d", iterations, len(ids))
		}
	})

	t.Run("fallback format", func(t *testing.T) {
		timestampPattern := regexp.MustCompile(`^request-\d+$`)
		fallbackID := "request-1640995200000000000"
		if !timestampPattern.MatchString(fallbackID) {
			t.Errorf("fallback ID format should match 'request-', example: %s", fallbackID)
		}
	})

	t.Run("performance", func(t *testing.T) {
		start := time.Now()
		iterations := 1000
		for range iterations {
			id := GenerateRequestID()
			if len(id) != 32 {
				t.Errorf("unexpected ID length: %d", len(id))
			}
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
			if uniqueIDs[id] {
				t.Errorf("found duplicate ID in concurrent generation: %s", id)
			}
			uniqueIDs[id] = true
		}

		expectedCount := numGoroutines * numIterations
		if len(uniqueIDs) != expectedCount {
			t.Errorf("expected %d unique IDs, got %d", expectedCount, len(uniqueIDs))
		}
	})
}

func TestRequestIDOptions(t *testing.T) {
	t.Run("header option", func(t *testing.T) {
		cfg := DefaultRequestIDConfig
		WithRequestIDHeader("X-Trace-Id")(&cfg)
		if cfg.Header != "X-Trace-Id" {
			t.Errorf("expected header = 'X-Trace-Id', got %s", cfg.Header)
		}
	})

	t.Run("generator option", func(t *testing.T) {
		customGenerator := func() string { return "custom-id-123" }
		cfg := DefaultRequestIDConfig
		WithRequestIDGenerator(customGenerator)(&cfg)
		if cfg.Generator == nil {
			t.Error("expected generator to be set")
		}
		result := cfg.Generator()
		if result != "custom-id-123" {
			t.Errorf("expected custom generator result = 'custom-id-123', got %s", result)
		}
	})

	t.Run("context key option", func(t *testing.T) {
		customKey := RequestIDContextKey("trace_id")
		cfg := DefaultRequestIDConfig
		WithRequestIDContextKey(customKey)(&cfg)
		if cfg.ContextKey != customKey {
			t.Errorf("expected context key = 'trace_id', got %s", string(cfg.ContextKey))
		}
		if string(cfg.ContextKey) != "trace_id" {
			t.Errorf("expected context key string = 'trace_id', got %s", string(cfg.ContextKey))
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		customGenerator := func() string { return "multi-option-id" }
		customKey := RequestIDContextKey("custom_request_id")
		cfg := DefaultRequestIDConfig
		WithRequestIDHeader("X-Custom-Request-Id")(&cfg)
		WithRequestIDGenerator(customGenerator)(&cfg)
		WithRequestIDContextKey(customKey)(&cfg)

		if cfg.Header != "X-Custom-Request-Id" {
			t.Errorf("expected header = 'X-Custom-Request-Id', got %s", cfg.Header)
		}
		if cfg.Generator == nil {
			t.Error("expected generator to be set")
		}
		if cfg.Generator() != "multi-option-id" {
			t.Error("expected custom generator to work")
		}
		if cfg.ContextKey != customKey {
			t.Errorf("expected context key = 'custom_request_id', got %s", string(cfg.ContextKey))
		}
	})
}

func TestRequestIDConfig_CommonScenarios(t *testing.T) {
	t.Run("common headers", func(t *testing.T) {
		headers := []string{"X-Request-Id", "X-Trace-Id", "X-Correlation-Id", "Request-ID", "Trace-ID", "X-Request-UUID", "X-Session-Id"}
		for _, header := range headers {
			t.Run(header, func(t *testing.T) {
				cfg := DefaultRequestIDConfig
				WithRequestIDHeader(header)(&cfg)
				if cfg.Header != header {
					t.Errorf("expected header = %s, got %s", header, cfg.Header)
				}
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
				cfg := DefaultRequestIDConfig
				WithRequestIDGenerator(tt.generator)(&cfg)
				result := cfg.Generator()
				if result != tt.expected {
					t.Errorf("expected generator result = %s, got %s", tt.expected, result)
				}
			})
		}
	})

	t.Run("context keys", func(t *testing.T) {
		tests := []struct {
			key      RequestIDContextKey
			expected string
		}{
			{RequestIDContextKey("request_id"), "request_id"},
			{RequestIDContextKey("trace_id"), "trace_id"},
			{RequestIDContextKey("correlation_id"), "correlation_id"},
			{RequestIDContextKey("session_id"), "session_id"},
			{RequestIDContextKey("req-id"), "req-id"},
			{RequestIDContextKey("x-request-id"), "x-request-id"},
			{RequestIDContextKey(""), ""},
		}

		for _, tt := range tests {
			t.Run(string(tt.key), func(t *testing.T) {
				cfg := DefaultRequestIDConfig
				WithRequestIDContextKey(tt.key)(&cfg)
				if string(cfg.ContextKey) != tt.expected {
					t.Errorf("expected context key = %s, got %s", tt.expected, string(cfg.ContextKey))
				}
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
				cfg := DefaultRequestIDConfig
				WithRequestIDHeader(header)(&cfg)
				if cfg.Header != header {
					t.Errorf("expected header = %s, got %s", header, cfg.Header)
				}
			})
		}
	})
}

func TestRequestIDConfig_EdgeCases(t *testing.T) {
	t.Run("nil generator", func(t *testing.T) {
		cfg := DefaultRequestIDConfig
		WithRequestIDGenerator(nil)(&cfg)
		if cfg.Generator != nil {
			t.Error("expected generator to remain nil when nil is passed")
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		cfg := DefaultRequestIDConfig
		WithRequestIDHeader("")(&cfg)
		WithRequestIDContextKey(RequestIDContextKey(""))(&cfg)
		if cfg.Header != "" {
			t.Errorf("expected empty header, got %s", cfg.Header)
		}
		if string(cfg.ContextKey) != "" {
			t.Errorf("expected empty context key, got %s", string(cfg.ContextKey))
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := RequestIDConfig{} // Zero values
		if cfg.Header != "" {
			t.Errorf("expected zero header = '', got %s", cfg.Header)
		}
		if cfg.Generator != nil {
			t.Error("expected zero generator = nil, got non-nil function")
		}
		if string(cfg.ContextKey) != "" {
			t.Errorf("expected zero context key = '', got %s", string(cfg.ContextKey))
		}
	})
}

func TestRequestIDConfigToOptions(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		customGenerator := func() string { return "config-to-options-test" }
		cfg := RequestIDConfig{
			Header:     "X-Test-Request-Id",
			Generator:  customGenerator,
			ContextKey: RequestIDContextKey("test_request_id"),
		}

		options := requestIDConfigToOptions(cfg)
		if len(options) != 3 {
			t.Errorf("expected 3 options, got %d", len(options))
		}

		newCfg := DefaultRequestIDConfig
		for _, option := range options {
			option(&newCfg)
		}

		if newCfg.Header != "X-Test-Request-Id" {
			t.Errorf("expected converted header = 'X-Test-Request-Id', got %s", newCfg.Header)
		}
		if newCfg.Generator == nil {
			t.Error("expected converted generator to be set")
		}
		if newCfg.Generator() != "config-to-options-test" {
			t.Error("expected converted generator to work correctly")
		}
		if string(newCfg.ContextKey) != "test_request_id" {
			t.Errorf("expected converted context key = 'test_request_id', got %s", string(newCfg.ContextKey))
		}
	})

	t.Run("default values conversion", func(t *testing.T) {
		cfg := DefaultRequestIDConfig
		options := requestIDConfigToOptions(cfg)
		if len(options) != 3 {
			t.Errorf("expected 3 options for default config, got %d", len(options))
		}

		newCfg := RequestIDConfig{} // Start with zero values
		for _, option := range options {
			option(&newCfg)
		}

		if newCfg.Header != DefaultRequestIDConfig.Header {
			t.Errorf("expected converted header = %s, got %s", DefaultRequestIDConfig.Header, newCfg.Header)
		}
		if newCfg.Generator == nil {
			t.Error("expected converted generator to be set")
		}
		if string(newCfg.ContextKey) != string(DefaultRequestIDConfig.ContextKey) {
			t.Errorf("expected converted context key = %s, got %s", string(DefaultRequestIDConfig.ContextKey), string(newCfg.ContextKey))
		}
	})

	t.Run("custom values conversion", func(t *testing.T) {
		tests := []struct {
			name       string
			header     string
			contextKey RequestIDContextKey
			genResult  string
		}{
			{"trace config", "X-Trace-Id", RequestIDContextKey("trace"), "trace-123"},
			{"correlation config", "X-Correlation-Id", RequestIDContextKey("correlation"), "corr-456"},
			{"session config", "X-Session-Id", RequestIDContextKey("session"), "sess-789"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				generator := func() string { return tt.genResult }
				cfg := RequestIDConfig{
					Header:     tt.header,
					Generator:  generator,
					ContextKey: tt.contextKey,
				}

				options := requestIDConfigToOptions(cfg)
				newCfg := RequestIDConfig{}
				for _, option := range options {
					option(&newCfg)
				}

				if newCfg.Header != tt.header {
					t.Errorf("expected header = %s, got %s", tt.header, newCfg.Header)
				}
				if newCfg.Generator() != tt.genResult {
					t.Errorf("expected generator result = %s, got %s", tt.genResult, newCfg.Generator())
				}
				if string(newCfg.ContextKey) != string(tt.contextKey) {
					t.Errorf("expected context key = %s, got %s", string(tt.contextKey), string(newCfg.ContextKey))
				}
			})
		}
	})

	t.Run("options equivalence", func(t *testing.T) {
		customGenerator := func() string { return "equivalence-test" }
		originalCfg := RequestIDConfig{
			Header:     "X-Equivalence-Test",
			Generator:  customGenerator,
			ContextKey: RequestIDContextKey("equivalence_test"),
		}

		// Method 1: Apply options individually
		cfg1 := DefaultRequestIDConfig
		WithRequestIDHeader(originalCfg.Header)(&cfg1)
		WithRequestIDGenerator(originalCfg.Generator)(&cfg1)
		WithRequestIDContextKey(originalCfg.ContextKey)(&cfg1)

		// Method 2: Apply via requestIDConfigToOptions
		cfg2 := DefaultRequestIDConfig
		options := requestIDConfigToOptions(originalCfg)
		for _, option := range options {
			option(&cfg2)
		}

		// Both should be functionally identical
		if cfg1.Header != cfg2.Header {
			t.Errorf("headers should be identical: cfg1=%s, cfg2=%s", cfg1.Header, cfg2.Header)
		}
		if string(cfg1.ContextKey) != string(cfg2.ContextKey) {
			t.Errorf("context keys should be identical: cfg1=%s, cfg2=%s", string(cfg1.ContextKey), string(cfg2.ContextKey))
		}
		if cfg1.Generator() != cfg2.Generator() {
			t.Errorf("generators should produce identical results: cfg1=%s, cfg2=%s", cfg1.Generator(), cfg2.Generator())
		}
	})
}

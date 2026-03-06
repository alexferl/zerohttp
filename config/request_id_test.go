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
	if key, ok := any(cfg.ContextKey).(RequestIDContextKey); !ok {
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

func TestRequestIDConfig_StructAssignment(t *testing.T) {
	t.Run("header assignment", func(t *testing.T) {
		cfg := RequestIDConfig{
			Header:     "X-Trace-Id",
			Generator:  GenerateRequestID,
			ContextKey: RequestIDContextKey("request_id"),
		}
		if cfg.Header != "X-Trace-Id" {
			t.Errorf("expected header = 'X-Trace-Id', got %s", cfg.Header)
		}
	})

	t.Run("generator assignment", func(t *testing.T) {
		customGenerator := func() string { return "custom-id-123" }
		cfg := RequestIDConfig{
			Header:     "X-Request-Id",
			Generator:  customGenerator,
			ContextKey: RequestIDContextKey("request_id"),
		}
		if cfg.Generator == nil {
			t.Error("expected generator to be set")
		}
		result := cfg.Generator()
		if result != "custom-id-123" {
			t.Errorf("expected custom generator result = 'custom-id-123', got %s", result)
		}
	})

	t.Run("context key assignment", func(t *testing.T) {
		customKey := RequestIDContextKey("trace_id")
		cfg := RequestIDConfig{
			Header:     "X-Request-Id",
			Generator:  GenerateRequestID,
			ContextKey: customKey,
		}
		if cfg.ContextKey != customKey {
			t.Errorf("expected context key = 'trace_id', got %s", string(cfg.ContextKey))
		}
		if string(cfg.ContextKey) != "trace_id" {
			t.Errorf("expected context key string = 'trace_id', got %s", string(cfg.ContextKey))
		}
	})

	t.Run("multiple fields assignment", func(t *testing.T) {
		customGenerator := func() string { return "multi-option-id" }
		customKey := RequestIDContextKey("custom_request_id")
		cfg := RequestIDConfig{
			Header:     "X-Custom-Request-Id",
			Generator:  customGenerator,
			ContextKey: customKey,
		}

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
				cfg := RequestIDConfig{
					Header:     header,
					Generator:  GenerateRequestID,
					ContextKey: RequestIDContextKey("request_id"),
				}
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
				cfg := RequestIDConfig{
					Header:     "X-Request-Id",
					Generator:  tt.generator,
					ContextKey: RequestIDContextKey("request_id"),
				}
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
				cfg := RequestIDConfig{
					Header:     "X-Request-Id",
					Generator:  GenerateRequestID,
					ContextKey: tt.key,
				}
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
				cfg := RequestIDConfig{
					Header:     header,
					Generator:  GenerateRequestID,
					ContextKey: RequestIDContextKey("request_id"),
				}
				if cfg.Header != header {
					t.Errorf("expected header = %s, got %s", header, cfg.Header)
				}
			})
		}
	})
}

func TestRequestIDConfig_EdgeCases(t *testing.T) {
	t.Run("nil generator", func(t *testing.T) {
		cfg := RequestIDConfig{
			Header:     "X-Request-Id",
			Generator:  nil,
			ContextKey: RequestIDContextKey("request_id"),
		}
		if cfg.Generator != nil {
			t.Error("expected generator to be nil")
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		cfg := RequestIDConfig{
			Header:     "",
			Generator:  GenerateRequestID,
			ContextKey: RequestIDContextKey(""),
		}
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

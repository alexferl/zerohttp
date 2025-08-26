package config

import (
	"net/http"
	"testing"
	"time"
)

func TestCircuitBreakerConfig_DefaultValues(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig
	if cfg.FailureThreshold != 5 {
		t.Errorf("expected default failure threshold = 5, got %d", cfg.FailureThreshold)
	}
	if cfg.RecoveryTimeout != 30*time.Second {
		t.Errorf("expected default recovery timeout = 30s, got %v", cfg.RecoveryTimeout)
	}
	if cfg.SuccessThreshold != 3 {
		t.Errorf("expected default success threshold = 3, got %d", cfg.SuccessThreshold)
	}
	if cfg.IsFailure == nil {
		t.Error("expected default IsFailure function to be set")
	}
	if cfg.KeyExtractor == nil {
		t.Error("expected default KeyExtractor function to be set")
	}
	if cfg.OpenStatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected default open status code = %d, got %d", http.StatusServiceUnavailable, cfg.OpenStatusCode)
	}
	if cfg.OpenMessage != "Service temporarily unavailable" {
		t.Errorf("expected default open message = 'Service temporarily unavailable', got %s", cfg.OpenMessage)
	}
}

func TestCircuitBreakerConfig_DefaultFunctions(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig
	req, _ := http.NewRequest("GET", "/test", nil)

	t.Run("default IsFailure function", func(t *testing.T) {
		tests := []struct {
			statusCode int
			expected   bool
		}{
			{200, false},
			{201, false},
			{400, false},
			{404, false},
			{499, false},
			{500, true},
			{501, true},
			{502, true},
			{503, true},
			{599, true},
		}
		for _, tt := range tests {
			result := cfg.IsFailure(req, tt.statusCode)
			if result != tt.expected {
				t.Errorf("IsFailure(req, %d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		}
	})

	t.Run("default KeyExtractor function", func(t *testing.T) {
		tests := []struct {
			path, expected string
		}{
			{"/api/users", "/api/users"},
			{"/health", "/health"},
			{"/api/v1/products/123", "/api/v1/products/123"},
			{"/", "/"},
			{"", ""},
		}
		for _, tt := range tests {
			req, _ := http.NewRequest("GET", tt.path, nil)
			result := cfg.KeyExtractor(req)
			if result != tt.expected {
				t.Errorf("KeyExtractor(req with path %s) = %s, expected %s", tt.path, result, tt.expected)
			}
		}
	})
}

func TestCircuitBreakerOptions(t *testing.T) {
	t.Run("failure threshold", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerFailureThreshold(10)(&cfg)
		if cfg.FailureThreshold != 10 {
			t.Errorf("expected failure threshold = 10, got %d", cfg.FailureThreshold)
		}
	})

	t.Run("recovery timeout", func(t *testing.T) {
		timeout := 60 * time.Second
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerRecoveryTimeout(timeout)(&cfg)
		if cfg.RecoveryTimeout != timeout {
			t.Errorf("expected recovery timeout = %v, got %v", timeout, cfg.RecoveryTimeout)
		}
	})

	t.Run("success threshold", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerSuccessThreshold(5)(&cfg)
		if cfg.SuccessThreshold != 5 {
			t.Errorf("expected success threshold = 5, got %d", cfg.SuccessThreshold)
		}
	})

	t.Run("open status code", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerOpenStatusCode(http.StatusTooManyRequests)(&cfg)
		if cfg.OpenStatusCode != http.StatusTooManyRequests {
			t.Errorf("expected open status code = %d, got %d", http.StatusTooManyRequests, cfg.OpenStatusCode)
		}
	})

	t.Run("open message", func(t *testing.T) {
		message := "Circuit breaker is open"
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerOpenMessage(message)(&cfg)
		if cfg.OpenMessage != message {
			t.Errorf("expected open message = %s, got %s", message, cfg.OpenMessage)
		}
	})
}

func TestCircuitBreakerConfig_CustomFunctions(t *testing.T) {
	t.Run("custom IsFailure function", func(t *testing.T) {
		customIsFailure := func(r *http.Request, statusCode int) bool {
			return statusCode >= 400
		}
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerIsFailure(customIsFailure)(&cfg)
		if cfg.IsFailure == nil {
			t.Error("expected IsFailure function to be set")
		}
		req, _ := http.NewRequest("GET", "/test", nil)
		tests := []struct {
			statusCode int
			expected   bool
		}{
			{200, false}, {300, false}, {399, false}, {400, true}, {404, true}, {500, true},
		}
		for _, tt := range tests {
			result := cfg.IsFailure(req, tt.statusCode)
			if result != tt.expected {
				t.Errorf("custom IsFailure(req, %d) = %v, expected %v", tt.statusCode, result, tt.expected)
			}
		}
	})

	t.Run("custom KeyExtractor function", func(t *testing.T) {
		customKeyExtractor := func(r *http.Request) string {
			return r.Method + ":" + r.URL.Path
		}
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerKeyExtractor(customKeyExtractor)(&cfg)
		if cfg.KeyExtractor == nil {
			t.Error("expected KeyExtractor function to be set")
		}
		tests := []struct {
			method, path, expected string
		}{
			{"GET", "/users", "GET:/users"},
			{"POST", "/api/data", "POST:/api/data"},
			{"PUT", "/", "PUT:/"},
		}
		for _, tt := range tests {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			result := cfg.KeyExtractor(req)
			if result != tt.expected {
				t.Errorf("custom KeyExtractor(%s %s) = %s, expected %s", tt.method, tt.path, result, tt.expected)
			}
		}
	})
}

func TestCircuitBreakerConfig_MultipleOptions(t *testing.T) {
	timeout := 45 * time.Second
	customIsFailure := func(r *http.Request, statusCode int) bool {
		return statusCode >= 400
	}
	customKeyExtractor := func(r *http.Request) string {
		return r.Host + r.URL.Path
	}

	cfg := DefaultCircuitBreakerConfig
	WithCircuitBreakerFailureThreshold(8)(&cfg)
	WithCircuitBreakerRecoveryTimeout(timeout)(&cfg)
	WithCircuitBreakerSuccessThreshold(4)(&cfg)
	WithCircuitBreakerIsFailure(customIsFailure)(&cfg)
	WithCircuitBreakerKeyExtractor(customKeyExtractor)(&cfg)
	WithCircuitBreakerOpenStatusCode(http.StatusTooManyRequests)(&cfg)
	WithCircuitBreakerOpenMessage("Service overloaded")(&cfg)

	if cfg.FailureThreshold != 8 {
		t.Errorf("expected failure threshold = 8, got %d", cfg.FailureThreshold)
	}
	if cfg.RecoveryTimeout != timeout {
		t.Errorf("expected recovery timeout = %v, got %v", timeout, cfg.RecoveryTimeout)
	}
	if cfg.SuccessThreshold != 4 {
		t.Errorf("expected success threshold = 4, got %d", cfg.SuccessThreshold)
	}
	if cfg.OpenStatusCode != http.StatusTooManyRequests {
		t.Errorf("expected open status code = %d, got %d", http.StatusTooManyRequests, cfg.OpenStatusCode)
	}
	if cfg.OpenMessage != "Service overloaded" {
		t.Errorf("expected open message = 'Service overloaded', got %s", cfg.OpenMessage)
	}

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Host = "example.com"
	if !cfg.IsFailure(req, 400) {
		t.Error("expected custom IsFailure to return true for 400")
	}
	if cfg.IsFailure(req, 200) {
		t.Error("expected custom IsFailure to return false for 200")
	}
	expectedKey := "example.com/test"
	if cfg.KeyExtractor(req) != expectedKey {
		t.Errorf("expected custom KeyExtractor to return %s, got %s", expectedKey, cfg.KeyExtractor(req))
	}
}

func TestCircuitBreakerConfig_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerFailureThreshold(0)(&cfg)
		WithCircuitBreakerRecoveryTimeout(0)(&cfg)
		WithCircuitBreakerSuccessThreshold(0)(&cfg)
		if cfg.FailureThreshold != 0 {
			t.Errorf("expected failure threshold = 0, got %d", cfg.FailureThreshold)
		}
		if cfg.RecoveryTimeout != 0 {
			t.Errorf("expected recovery timeout = 0, got %v", cfg.RecoveryTimeout)
		}
		if cfg.SuccessThreshold != 0 {
			t.Errorf("expected success threshold = 0, got %d", cfg.SuccessThreshold)
		}
	})

	t.Run("nil functions", func(t *testing.T) {
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerIsFailure(nil)(&cfg)
		WithCircuitBreakerKeyExtractor(nil)(&cfg)
		if cfg.IsFailure != nil {
			t.Error("expected IsFailure to be nil when nil is passed")
		}
		if cfg.KeyExtractor != nil {
			t.Error("expected KeyExtractor to be nil when nil is passed")
		}
	})
}

func TestCircuitBreakerConfig_ComplexFunctionality(t *testing.T) {
	t.Run("path-based failure logic", func(t *testing.T) {
		customIsFailure := func(r *http.Request, statusCode int) bool {
			if r.URL.Path == "/critical" {
				return statusCode == 408 || statusCode >= 500
			}
			return statusCode >= 500
		}
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerIsFailure(customIsFailure)(&cfg)
		tests := []struct {
			path       string
			statusCode int
			expected   bool
		}{
			{"/critical", 200, false},
			{"/critical", 404, false},
			{"/critical", 408, true},
			{"/critical", 500, true},
			{"/normal", 200, false},
			{"/normal", 404, false},
			{"/normal", 408, false},
			{"/normal", 500, true},
		}
		for _, tt := range tests {
			req, _ := http.NewRequest("GET", tt.path, nil)
			result := cfg.IsFailure(req, tt.statusCode)
			if result != tt.expected {
				t.Errorf("custom IsFailure(req with path %s, %d) = %v, expected %v", tt.path, tt.statusCode, result, tt.expected)
			}
		}
	})

	t.Run("user-aware key extraction", func(t *testing.T) {
		customKeyExtractor := func(r *http.Request) string {
			userID := r.Header.Get("X-User-ID")
			if userID != "" {
				return userID + ":" + r.URL.Path
			}
			return r.URL.Path
		}
		cfg := DefaultCircuitBreakerConfig
		WithCircuitBreakerKeyExtractor(customKeyExtractor)(&cfg)
		tests := []struct {
			path, userID, expected string
		}{
			{"/api/data", "", "/api/data"},
			{"/api/data", "user123", "user123:/api/data"},
			{"/health", "admin", "admin:/health"},
			{"/", "", "/"},
		}
		for _, tt := range tests {
			req, _ := http.NewRequest("GET", tt.path, nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			result := cfg.KeyExtractor(req)
			if result != tt.expected {
				t.Errorf("custom KeyExtractor(req with path %s, userID %s) = %s, expected %s", tt.path, tt.userID, result, tt.expected)
			}
		}
	})
}

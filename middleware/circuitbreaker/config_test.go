package circuitbreaker

import (
	"net/http"
	"testing"
	"time"
)

func TestCircuitBreakerConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
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
	cfg := DefaultConfig
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)

	t.Run("default IsFailure function", func(t *testing.T) {
		tests := []struct {
			statusCode int
			expected   bool
		}{
			{http.StatusOK, false},
			{http.StatusCreated, false},
			{http.StatusBadRequest, false},
			{http.StatusNotFound, false},
			{499, false},
			{http.StatusInternalServerError, true},
			{http.StatusNotImplemented, true},
			{http.StatusBadGateway, true},
			{http.StatusServiceUnavailable, true},
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
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			result := cfg.KeyExtractor(req)
			if result != tt.expected {
				t.Errorf("KeyExtractor(req with path %s) = %s, expected %s", tt.path, result, tt.expected)
			}
		}
	})
}

func TestCircuitBreakerConfig_StructAssignment(t *testing.T) {
	t.Run("failure threshold", func(t *testing.T) {
		cfg := Config{
			FailureThreshold: 10,
		}
		if cfg.FailureThreshold != 10 {
			t.Errorf("expected failure threshold = 10, got %d", cfg.FailureThreshold)
		}
	})

	t.Run("recovery timeout", func(t *testing.T) {
		timeout := 60 * time.Second
		cfg := Config{
			RecoveryTimeout: timeout,
		}
		if cfg.RecoveryTimeout != timeout {
			t.Errorf("expected recovery timeout = %v, got %v", timeout, cfg.RecoveryTimeout)
		}
	})

	t.Run("success threshold", func(t *testing.T) {
		cfg := Config{
			SuccessThreshold: 5,
		}
		if cfg.SuccessThreshold != 5 {
			t.Errorf("expected success threshold = 5, got %d", cfg.SuccessThreshold)
		}
	})

	t.Run("open status code", func(t *testing.T) {
		cfg := Config{
			OpenStatusCode: http.StatusTooManyRequests,
		}
		if cfg.OpenStatusCode != http.StatusTooManyRequests {
			t.Errorf("expected open status code = %d, got %d", http.StatusTooManyRequests, cfg.OpenStatusCode)
		}
	})

	t.Run("open message", func(t *testing.T) {
		message := "Circuit breaker is open"
		cfg := Config{
			OpenMessage: message,
		}
		if cfg.OpenMessage != message {
			t.Errorf("expected open message = %s, got %s", message, cfg.OpenMessage)
		}
	})
}

func TestCircuitBreakerConfig_CustomFunctions(t *testing.T) {
	t.Run("custom IsFailure function", func(t *testing.T) {
		customIsFailure := func(r *http.Request, statusCode int) bool {
			return statusCode >= http.StatusBadRequest
		}
		cfg := Config{
			IsFailure: customIsFailure,
		}
		if cfg.IsFailure == nil {
			t.Error("expected IsFailure function to be set")
		}
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		tests := []struct {
			statusCode int
			expected   bool
		}{
			{http.StatusOK, false},
			{http.StatusMultipleChoices, false},
			{399, false},
			{http.StatusBadRequest, true},
			{http.StatusNotFound, true},
			{http.StatusInternalServerError, true},
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
		cfg := Config{
			KeyExtractor: customKeyExtractor,
		}
		if cfg.KeyExtractor == nil {
			t.Error("expected KeyExtractor function to be set")
		}
		tests := []struct {
			method, path, expected string
		}{
			{http.MethodGet, "/users", "GET:/users"},
			{http.MethodPost, "/api/data", "POST:/api/data"},
			{http.MethodPut, "/", "PUT:/"},
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

func TestCircuitBreakerConfig_MultipleFields(t *testing.T) {
	timeout := 45 * time.Second
	customIsFailure := func(r *http.Request, statusCode int) bool {
		return statusCode >= http.StatusBadRequest
	}
	customKeyExtractor := func(r *http.Request) string {
		return r.Host + r.URL.Path
	}

	cfg := Config{
		FailureThreshold: 8,
		RecoveryTimeout:  timeout,
		SuccessThreshold: 4,
		IsFailure:        customIsFailure,
		KeyExtractor:     customKeyExtractor,
		OpenStatusCode:   http.StatusTooManyRequests,
		OpenMessage:      "Service overloaded",
	}

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

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "example.com"
	if !cfg.IsFailure(req, http.StatusBadRequest) {
		t.Error("expected custom IsFailure to return true for 400")
	}
	if cfg.IsFailure(req, http.StatusOK) {
		t.Error("expected custom IsFailure to return false for 200")
	}
	expectedKey := "example.com/test"
	if cfg.KeyExtractor(req) != expectedKey {
		t.Errorf("expected custom KeyExtractor to return %s, got %s", expectedKey, cfg.KeyExtractor(req))
	}
}

func TestCircuitBreakerConfig_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cfg := Config{
			FailureThreshold: 0,
			RecoveryTimeout:  0,
			SuccessThreshold: 0,
		}
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
		cfg := Config{
			IsFailure:    nil,
			KeyExtractor: nil,
		}
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
				return statusCode == http.StatusRequestTimeout || statusCode >= http.StatusInternalServerError
			}
			return statusCode >= http.StatusInternalServerError
		}
		cfg := Config{
			IsFailure: customIsFailure,
		}
		tests := []struct {
			path       string
			statusCode int
			expected   bool
		}{
			{"/critical", http.StatusOK, false},
			{"/critical", http.StatusNotFound, false},
			{"/critical", http.StatusRequestTimeout, true},
			{"/critical", http.StatusInternalServerError, true},
			{"/normal", http.StatusOK, false},
			{"/normal", http.StatusNotFound, false},
			{"/normal", http.StatusRequestTimeout, false},
			{"/normal", http.StatusInternalServerError, true},
		}
		for _, tt := range tests {
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
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
		cfg := Config{
			KeyExtractor: customKeyExtractor,
		}
		tests := []struct {
			path, userID, expected string
		}{
			{"/api/data", "", "/api/data"},
			{"/api/data", "user123", "user123:/api/data"},
			{"/health", "admin", "admin:/health"},
			{"/", "", "/"},
		}
		for _, tt := range tests {
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
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

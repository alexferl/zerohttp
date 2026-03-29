package circuitbreaker

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestCircuitBreakerConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 5, cfg.FailureThreshold)
	zhtest.AssertEqual(t, 30*time.Second, cfg.RecoveryTimeout)
	zhtest.AssertEqual(t, 3, cfg.SuccessThreshold)
	zhtest.AssertNotNil(t, cfg.IsFailure)
	zhtest.AssertNotNil(t, cfg.KeyExtractor)
	zhtest.AssertEqual(t, http.StatusServiceUnavailable, cfg.OpenStatusCode)
	zhtest.AssertEqual(t, "Service temporarily unavailable", cfg.OpenMessage)
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
			zhtest.AssertEqual(t, tt.expected, result)
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
			zhtest.AssertEqual(t, tt.expected, result)
		}
	})
}

func TestCircuitBreakerConfig_StructAssignment(t *testing.T) {
	t.Run("failure threshold", func(t *testing.T) {
		cfg := Config{
			FailureThreshold: 10,
		}
		zhtest.AssertEqual(t, 10, cfg.FailureThreshold)
	})

	t.Run("recovery timeout", func(t *testing.T) {
		timeout := 60 * time.Second
		cfg := Config{
			RecoveryTimeout: timeout,
		}
		zhtest.AssertEqual(t, timeout, cfg.RecoveryTimeout)
	})

	t.Run("success threshold", func(t *testing.T) {
		cfg := Config{
			SuccessThreshold: 5,
		}
		zhtest.AssertEqual(t, 5, cfg.SuccessThreshold)
	})

	t.Run("open status code", func(t *testing.T) {
		cfg := Config{
			OpenStatusCode: http.StatusTooManyRequests,
		}
		zhtest.AssertEqual(t, http.StatusTooManyRequests, cfg.OpenStatusCode)
	})

	t.Run("open message", func(t *testing.T) {
		message := "Circuit breaker is open"
		cfg := Config{
			OpenMessage: message,
		}
		zhtest.AssertEqual(t, message, cfg.OpenMessage)
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
		zhtest.AssertNotNil(t, cfg.IsFailure)
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
			zhtest.AssertEqual(t, tt.expected, result)
		}
	})

	t.Run("custom KeyExtractor function", func(t *testing.T) {
		customKeyExtractor := func(r *http.Request) string {
			return r.Method + ":" + r.URL.Path
		}
		cfg := Config{
			KeyExtractor: customKeyExtractor,
		}
		zhtest.AssertNotNil(t, cfg.KeyExtractor)
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
			zhtest.AssertEqual(t, tt.expected, result)
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

	zhtest.AssertEqual(t, 8, cfg.FailureThreshold)
	zhtest.AssertEqual(t, timeout, cfg.RecoveryTimeout)
	zhtest.AssertEqual(t, 4, cfg.SuccessThreshold)
	zhtest.AssertEqual(t, http.StatusTooManyRequests, cfg.OpenStatusCode)
	zhtest.AssertEqual(t, "Service overloaded", cfg.OpenMessage)

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Host = "example.com"
	zhtest.AssertTrue(t, cfg.IsFailure(req, http.StatusBadRequest))
	zhtest.AssertFalse(t, cfg.IsFailure(req, http.StatusOK))
	zhtest.AssertEqual(t, "example.com/test", cfg.KeyExtractor(req))
}

func TestCircuitBreakerConfig_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cfg := Config{
			FailureThreshold: 0,
			RecoveryTimeout:  0,
			SuccessThreshold: 0,
		}
		zhtest.AssertEqual(t, 0, cfg.FailureThreshold)
		zhtest.AssertEqual(t, time.Duration(0), cfg.RecoveryTimeout)
		zhtest.AssertEqual(t, 0, cfg.SuccessThreshold)
	})

	t.Run("nil functions", func(t *testing.T) {
		cfg := Config{
			IsFailure:    nil,
			KeyExtractor: nil,
		}
		zhtest.AssertNil(t, cfg.IsFailure)
		zhtest.AssertNil(t, cfg.KeyExtractor)
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
			zhtest.AssertEqual(t, tt.expected, result)
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
			zhtest.AssertEqual(t, tt.expected, result)
		}
	})
}

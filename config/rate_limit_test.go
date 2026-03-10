package config

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestRateLimitConfig_DefaultValues(t *testing.T) {
	cfg := DefaultRateLimitConfig
	if cfg.Rate != 100 {
		t.Errorf("expected default rate = 100, got %d", cfg.Rate)
	}
	if cfg.Window != time.Minute {
		t.Errorf("expected default window = 1m, got %v", cfg.Window)
	}
	if cfg.Algorithm != TokenBucket {
		t.Errorf("expected default algorithm = %s, got %s", TokenBucket, cfg.Algorithm)
	}
	if cfg.KeyExtractor == nil {
		t.Error("expected default key extractor to be set")
	}
	if cfg.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected default status code = %d, got %d", http.StatusTooManyRequests, cfg.StatusCode)
	}
	if cfg.Message != "Rate limit exceeded" {
		t.Errorf("expected default message = 'Rate limit exceeded', got %s", cfg.Message)
	}
	if cfg.IncludeHeaders != true {
		t.Errorf("expected default include headers = true, got %t", cfg.IncludeHeaders)
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}
}

func TestDefaultKeyExtractorFunction(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		expectedKey   string
	}{
		{"no forwarded header", "192.168.1.1:8080", "", "192.168.1.1"},
		{"with forwarded header", "192.168.1.1:8080", "203.0.113.1", "203.0.113.1"},
		{"empty forwarded header", "192.168.1.1:8080", "", "192.168.1.1"},
		{"forwarded with multiple IPs", "192.168.1.1:8080", "203.0.113.1, 198.51.100.1", "203.0.113.1"},
		{"IPv6 address", "[::1]:8080", "", "::1"},
		{"IPv6 with forwarded", "[::1]:8080", "2001:db8::1", "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			result := DefaultKeyExtractor(req)
			if result != tt.expectedKey {
				t.Errorf("expected key = %s, got %s", tt.expectedKey, result)
			}
		})
	}
}

func TestRateLimitConfig_StructAssignment(t *testing.T) {
	t.Run("basic fields", func(t *testing.T) {
		cfg := RateLimitConfig{
			Rate:           50,
			Window:         30 * time.Second,
			Algorithm:      SlidingWindow,
			StatusCode:     http.StatusServiceUnavailable,
			Message:        "Too many requests, please try again later",
			IncludeHeaders: false,
		}
		if cfg.Rate != 50 {
			t.Errorf("expected rate = 50, got %d", cfg.Rate)
		}
		if cfg.Window != 30*time.Second {
			t.Errorf("expected window = %v, got %v", 30*time.Second, cfg.Window)
		}
		if cfg.Algorithm != SlidingWindow {
			t.Errorf("expected algorithm = %s, got %s", SlidingWindow, cfg.Algorithm)
		}
		if cfg.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected status code = %d, got %d", http.StatusServiceUnavailable, cfg.StatusCode)
		}
		if cfg.Message != "Too many requests, please try again later" {
			t.Errorf("expected message = %s, got %s", "Too many requests, please try again later", cfg.Message)
		}
		if cfg.IncludeHeaders != false {
			t.Errorf("expected include headers = false, got %t", cfg.IncludeHeaders)
		}
	})

	t.Run("key extractor", func(t *testing.T) {
		customExtractor := func(r *http.Request) string {
			return r.Header.Get("X-User-ID")
		}
		cfg := RateLimitConfig{
			KeyExtractor: customExtractor,
		}
		if cfg.KeyExtractor == nil {
			t.Error("expected key extractor to be set")
		}

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-User-ID", "user123")
		result := cfg.KeyExtractor(req)
		if result != "user123" {
			t.Errorf("expected key = 'user123', got %s", result)
		}
	})

	t.Run("exempt paths", func(t *testing.T) {
		exemptPaths := []string{"/health", "/metrics", "/ping", "/status"}
		cfg := RateLimitConfig{
			ExemptPaths: exemptPaths,
		}
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("all algorithms", func(t *testing.T) {
		algorithms := []RateLimitAlgorithm{TokenBucket, SlidingWindow, FixedWindow}
		for _, algorithm := range algorithms {
			cfg := RateLimitConfig{
				Algorithm: algorithm,
			}
			if cfg.Algorithm != algorithm {
				t.Errorf("expected algorithm = %s, got %s", algorithm, cfg.Algorithm)
			}
		}
	})
}

func TestRateLimitConfig_MultipleFields(t *testing.T) {
	customExtractor := func(r *http.Request) string {
		return r.Header.Get("Authorization")
	}
	exemptPaths := []string{"/public", "/health"}
	cfg := RateLimitConfig{
		Rate:           200,
		Window:         5 * time.Minute,
		Algorithm:      FixedWindow,
		KeyExtractor:   customExtractor,
		StatusCode:     http.StatusForbidden,
		Message:        "Rate limit reached",
		IncludeHeaders: false,
		ExemptPaths:    exemptPaths,
	}

	if cfg.Rate != 200 {
		t.Errorf("expected rate = 200, got %d", cfg.Rate)
	}
	if cfg.Window != 5*time.Minute {
		t.Errorf("expected window = 5m, got %v", cfg.Window)
	}
	if cfg.Algorithm != FixedWindow {
		t.Errorf("expected algorithm = %s, got %s", FixedWindow, cfg.Algorithm)
	}
	if cfg.KeyExtractor == nil {
		t.Error("expected key extractor to be set")
	}
	if cfg.StatusCode != http.StatusForbidden {
		t.Errorf("expected status code = %d, got %d", http.StatusForbidden, cfg.StatusCode)
	}
	if cfg.Message != "Rate limit reached" {
		t.Errorf("expected message = 'Rate limit reached', got %s", cfg.Message)
	}
	if cfg.IncludeHeaders != false {
		t.Errorf("expected include headers = false, got %t", cfg.IncludeHeaders)
	}
	if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
		t.Error("expected exempt paths to be set correctly")
	}

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	if cfg.KeyExtractor(req) != "Bearer token123" {
		t.Error("expected custom key extractor to work")
	}
}

func TestRateLimitConfig_EdgeCases(t *testing.T) {
	t.Run("rate boundary values", func(t *testing.T) {
		testCases := []int{1, 10, 100, 1000, 0, -1}
		for _, rate := range testCases {
			cfg := RateLimitConfig{
				Rate: rate,
			}
			if cfg.Rate != rate {
				t.Errorf("Rate %d: expected rate = %d, got %d", rate, rate, cfg.Rate)
			}
		}
	})

	t.Run("window boundary values", func(t *testing.T) {
		testCases := []time.Duration{time.Second, 30 * time.Second, time.Minute, 5 * time.Minute, time.Hour, 0}
		for _, window := range testCases {
			cfg := RateLimitConfig{
				Window: window,
			}
			if cfg.Window != window {
				t.Errorf("Window %v: expected window = %v, got %v", window, window, cfg.Window)
			}
		}
	})

	t.Run("status code options", func(t *testing.T) {
		testCases := []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusForbidden, http.StatusBadRequest, http.StatusInternalServerError, 0}
		for _, statusCode := range testCases {
			cfg := RateLimitConfig{
				StatusCode: statusCode,
			}
			if cfg.StatusCode != statusCode {
				t.Errorf("StatusCode %d: expected %d, got %d", statusCode, statusCode, cfg.StatusCode)
			}
		}
	})

	t.Run("message options", func(t *testing.T) {
		cfg := RateLimitConfig{
			Message: "",
		}
		if cfg.Message != "" {
			t.Errorf("expected empty message, got %s", cfg.Message)
		}

		longMessage := "This is a very long rate limit message that explains in detail why the request was rejected and what the client should do to resolve the issue including waiting for the rate limit window to reset."
		cfg2 := RateLimitConfig{
			Message: longMessage,
		}
		if cfg2.Message != longMessage {
			t.Errorf("expected long message to be preserved")
		}
	})

	t.Run("empty and nil exempt paths", func(t *testing.T) {
		cfg := RateLimitConfig{
			ExemptPaths: []string{},
		}
		if cfg.ExemptPaths == nil || len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %v", cfg.ExemptPaths)
		}

		cfg2 := RateLimitConfig{
			ExemptPaths: nil,
		}
		if cfg2.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("nil key extractor", func(t *testing.T) {
		cfg := RateLimitConfig{
			KeyExtractor: nil,
		}
		if cfg.KeyExtractor != nil {
			t.Error("expected key extractor to remain nil when nil is passed")
		}
	})
}

func TestRateLimitConfig_CustomKeyExtractors(t *testing.T) {
	tests := []struct {
		name         string
		extractor    KeyExtractor
		setupRequest func(*http.Request)
		expectedKey  string
	}{
		{
			name: "user ID extractor",
			extractor: func(r *http.Request) string {
				return r.Header.Get("X-User-ID")
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-User-ID", "user456")
			},
			expectedKey: "user456",
		},
		{
			name: "API key extractor",
			extractor: func(r *http.Request) string {
				return r.Header.Get("X-API-Key")
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "key789")
			},
			expectedKey: "key789",
		},
		{
			name: "path-based extractor",
			extractor: func(r *http.Request) string {
				return r.URL.Path
			},
			setupRequest: func(r *http.Request) {
				r.URL.Path = "/api/users"
			},
			expectedKey: "/api/users",
		},
		{
			name: "combined extractor",
			extractor: func(r *http.Request) string {
				user := r.Header.Get("X-User-ID")
				if user == "" {
					return r.RemoteAddr
				}
				return user + ":" + r.URL.Path
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-User-ID", "user123")
				r.URL.Path = "/api/data"
				r.RemoteAddr = "192.168.1.1:8080"
			},
			expectedKey: "user123:/api/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := RateLimitConfig{
				KeyExtractor: tt.extractor,
			}
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			tt.setupRequest(req)
			result := cfg.KeyExtractor(req)
			if result != tt.expectedKey {
				t.Errorf("expected key = %s, got %s", tt.expectedKey, result)
			}
		})
	}
}

func TestRateLimitConfig_PathPatterns(t *testing.T) {
	exemptPaths := []string{
		"/health",
		"/metrics",
		"/api/v1/health/*",
		"/static/*",
		"*.json",
		"/admin/debug/*",
		"/internal/status",
	}
	cfg := RateLimitConfig{
		ExemptPaths: exemptPaths,
	}
	if len(cfg.ExemptPaths) != len(exemptPaths) {
		t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
	}
	if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
		t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
	}
}

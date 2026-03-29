package ratelimit

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRateLimitConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 100, cfg.Rate)
	zhtest.AssertEqual(t, time.Minute, cfg.Window)
	zhtest.AssertEqual(t, TokenBucket, cfg.Algorithm)
	zhtest.AssertNil(t, cfg.KeyExtractor)
	zhtest.AssertEqual(t, http.StatusTooManyRequests, cfg.StatusCode)
	zhtest.AssertEqual(t, "Rate limit exceeded", cfg.Message)
	zhtest.AssertTrue(t, cfg.IncludeHeaders)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
}

func TestRateLimitConfig_StructAssignment(t *testing.T) {
	t.Run("basic fields", func(t *testing.T) {
		cfg := Config{
			Rate:           50,
			Window:         30 * time.Second,
			Algorithm:      SlidingWindow,
			StatusCode:     http.StatusServiceUnavailable,
			Message:        "Too many requests, please try again later",
			IncludeHeaders: false,
		}
		zhtest.AssertEqual(t, 50, cfg.Rate)
		zhtest.AssertEqual(t, 30*time.Second, cfg.Window)
		zhtest.AssertEqual(t, SlidingWindow, cfg.Algorithm)
		zhtest.AssertEqual(t, http.StatusServiceUnavailable, cfg.StatusCode)
		zhtest.AssertEqual(t, "Too many requests, please try again later", cfg.Message)
		zhtest.AssertFalse(t, cfg.IncludeHeaders)
	})

	t.Run("key extractor", func(t *testing.T) {
		customExtractor := func(r *http.Request) string {
			return r.Header.Get("X-User-ID")
		}
		cfg := Config{
			KeyExtractor: customExtractor,
		}
		zhtest.AssertNotNil(t, cfg.KeyExtractor)

		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-User-ID", "user123")
		zhtest.AssertEqual(t, "user123", cfg.KeyExtractor(req))
	})

	t.Run("excluded paths", func(t *testing.T) {
		excludedPaths := []string{"/health", "/metrics", "/ping", "/status"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, includedPaths, cfg.IncludedPaths)
	})

	t.Run("all algorithms", func(t *testing.T) {
		algorithms := []Algorithm{TokenBucket, SlidingWindow, FixedWindow}
		for _, algorithm := range algorithms {
			cfg := Config{
				Algorithm: algorithm,
			}
			zhtest.AssertEqual(t, algorithm, cfg.Algorithm)
		}
	})
}

func TestRateLimitConfig_MultipleFields(t *testing.T) {
	customExtractor := func(r *http.Request) string {
		return r.Header.Get(httpx.HeaderAuthorization)
	}
	excludedPaths := []string{"/public", "/health"}
	includedPaths := []string{"/api/public"}
	cfg := Config{
		Rate:           200,
		Window:         5 * time.Minute,
		Algorithm:      FixedWindow,
		KeyExtractor:   customExtractor,
		StatusCode:     http.StatusForbidden,
		Message:        "Rate limit reached",
		IncludeHeaders: false,
		ExcludedPaths:  excludedPaths,
		IncludedPaths:  includedPaths,
	}

	zhtest.AssertEqual(t, 200, cfg.Rate)
	zhtest.AssertEqual(t, 5*time.Minute, cfg.Window)
	zhtest.AssertEqual(t, FixedWindow, cfg.Algorithm)
	zhtest.AssertNotNil(t, cfg.KeyExtractor)
	zhtest.AssertEqual(t, http.StatusForbidden, cfg.StatusCode)
	zhtest.AssertEqual(t, "Rate limit reached", cfg.Message)
	zhtest.AssertFalse(t, cfg.IncludeHeaders)
	zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
	zhtest.AssertEqual(t, includedPaths, cfg.IncludedPaths)

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer token123")
	zhtest.AssertEqual(t, "Bearer token123", cfg.KeyExtractor(req))
}

func TestRateLimitConfig_EdgeCases(t *testing.T) {
	t.Run("rate boundary values", func(t *testing.T) {
		testCases := []int{1, 10, 100, 1000, 0, -1}
		for _, rate := range testCases {
			cfg := Config{
				Rate: rate,
			}
			zhtest.AssertEqual(t, rate, cfg.Rate)
		}
	})

	t.Run("window boundary values", func(t *testing.T) {
		testCases := []time.Duration{time.Second, 30 * time.Second, time.Minute, 5 * time.Minute, time.Hour, 0}
		for _, window := range testCases {
			cfg := Config{
				Window: window,
			}
			zhtest.AssertEqual(t, window, cfg.Window)
		}
	})

	t.Run("status code options", func(t *testing.T) {
		testCases := []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusForbidden, http.StatusBadRequest, http.StatusInternalServerError, 0}
		for _, statusCode := range testCases {
			cfg := Config{
				StatusCode: statusCode,
			}
			zhtest.AssertEqual(t, statusCode, cfg.StatusCode)
		}
	})

	t.Run("message options", func(t *testing.T) {
		cfg := Config{
			Message: "",
		}
		zhtest.AssertEqual(t, "", cfg.Message)

		longMessage := "This is a very long rate limit message that explains in detail why the request was rejected and what the client should do to resolve the issue including waiting for the rate limit window to reset."
		cfg2 := Config{
			Message: longMessage,
		}
		zhtest.AssertEqual(t, longMessage, cfg2.Message)
	})

	t.Run("empty and nil excluded paths", func(t *testing.T) {
		cfg := Config{
			ExcludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))

		cfg2 := Config{
			ExcludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg2.ExcludedPaths)
	})

	t.Run("empty and nil included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

		cfg2 := Config{
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg2.IncludedPaths)
	})

	t.Run("nil key extractor", func(t *testing.T) {
		cfg := Config{
			KeyExtractor: nil,
		}
		zhtest.AssertNil(t, cfg.KeyExtractor)
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
			cfg := Config{
				KeyExtractor: tt.extractor,
			}
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			tt.setupRequest(req)
			zhtest.AssertEqual(t, tt.expectedKey, cfg.KeyExtractor(req))
		})
	}
}

func TestRateLimitConfig_PathPatterns(t *testing.T) {
	excludedPaths := []string{
		"/health",
		"/metrics",
		"/api/v1/health/*",
		"/static/*",
		"*.json",
		"/admin/debug/*",
		"/internal/status",
	}
	cfg := Config{
		ExcludedPaths: excludedPaths,
	}
	zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
}

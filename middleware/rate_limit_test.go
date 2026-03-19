package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRateLimit_DefaultConfigFallbacks(t *testing.T) {
	tests := []struct {
		name   string
		option func() func(http.Handler) http.Handler
	}{
		{"rate", func() func(http.Handler) http.Handler { return RateLimit(config.RateLimitConfig{Rate: 0}) }},
		{"window", func() func(http.Handler) http.Handler { return RateLimit(config.RateLimitConfig{Window: 0}) }},
		{"algorithm", func() func(http.Handler) http.Handler { return RateLimit(config.RateLimitConfig{Algorithm: ""}) }},
		{"key extractor", func() func(http.Handler) http.Handler { return RateLimit(config.RateLimitConfig{KeyExtractor: nil}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.option()
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			w := zhtest.TestMiddleware(m, req)

			zhtest.AssertWith(t, w).Status(http.StatusOK)
		})
	}
}

func TestRateLimitStatusCodeAndMessageDefaults(t *testing.T) {
	m := RateLimit(config.RateLimitConfig{StatusCode: 0, Rate: 1, Window: time.Minute})
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(handler, req)
	// Test JSON response
	req = zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	w := zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).IsProblemDetail().ProblemDetailDetail("Rate limit exceeded")

	// Test plain text response
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
}

func TestRateLimitMessageDefaults(t *testing.T) {
	m := RateLimit(config.RateLimitConfig{Message: "", Rate: 1, Window: time.Minute})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := m(next)
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(handler, req) // 1st OK

	// Test JSON response
	req = zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	w := zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).IsProblemDetail().ProblemDetailDetail("Rate limit exceeded")

	// Test plain text response
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w = zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
}

func TestRateLimitTokenBucket(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:      2,
		Window:    time.Second,
		Algorithm: config.TokenBucket,
	})
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	for i := range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		req.RemoteAddr = "127.0.0.1:12345"
		w := zhtest.Serve(handler, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
	// Test JSON response
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).IsProblemDetail().ProblemDetailDetail("Rate limit exceeded")

	// Test plain text response
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w = zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")

	if count != 2 {
		t.Errorf("expected 2 successful requests, got %d", count)
	}
}

func TestRateLimitFixedWindow(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:      3,
		Window:    time.Second,
		Algorithm: config.FixedWindow,
	})
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	for i := range 3 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		req.RemoteAddr = "127.0.0.1:12345"
		w := zhtest.Serve(handler, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests)
	if count != 3 {
		t.Errorf("expected 3 successful requests, got %d", count)
	}
}

func TestRateLimitSlidingWindow(t *testing.T) {
	m := RateLimit(config.RateLimitConfig{
		Rate:      2,
		Window:    100 * time.Millisecond,
		Algorithm: config.SlidingWindow,
	})
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	for i := range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		req.RemoteAddr = "127.0.0.1:12345"
		w := zhtest.Serve(handler, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests)

	time.Sleep(110 * time.Millisecond)
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w = zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestRateLimitHeaders(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:           5,
		Window:         time.Minute,
		Algorithm:      config.TokenBucket,
		IncludeHeaders: true,
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusOK).
		Header("X-RateLimit-Limit", "5").
		HeaderExists("X-RateLimit-Remaining").
		HeaderExists("X-RateLimit-Reset").
		Header("X-RateLimit-Window", "1m0s")
}

func TestRateLimitNoHeaders(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:           2,
		Window:         time.Second,
		Algorithm:      config.TokenBucket,
		IncludeHeaders: false,
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	for _, hdr := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		zhtest.AssertWith(t, w).HeaderNotExists(hdr)
	}
}

func TestRateLimitCustomKeyExtractor(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:      2,
		Window:    time.Second,
		Algorithm: config.TokenBucket,
		KeyExtractor: func(r *http.Request) string {
			return r.Header.Get("User-ID")
		},
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	// User 1 allowed first two
	for i := range 2 {
		req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("User-ID", "user1").Build()
		w := zhtest.Serve(handler, req)
		if w.Code != http.StatusOK {
			t.Errorf("user1 request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("User-ID", "user1").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests)

	req = zhtest.NewRequest(http.MethodGet, "/test").WithHeader("User-ID", "user2").Build()
	w = zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestRateLimitExcludedPaths(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:          1,
		Window:        time.Second,
		Algorithm:     config.TokenBucket,
		ExcludedPaths: []string{"/health", "/metrics"},
	})
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	for i := range 3 {
		req := zhtest.NewRequest(http.MethodGet, "/health").Build()
		req.RemoteAddr = "127.0.0.1:12345"
		w := zhtest.Serve(handler, req)

		if w.Code != http.StatusOK {
			t.Errorf("excluded path request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
	if count != 3 {
		t.Errorf("expected 3 requests to excluded path, got %d", count)
	}
	req := zhtest.NewRequest(http.MethodGet, "/api").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	req = zhtest.NewRequest(http.MethodGet, "/api").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w = zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests)
}

func TestRateLimitCustomMessage(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:       1,
		Window:     time.Second,
		Algorithm:  config.TokenBucket,
		Message:    "Too many requests, please slow down",
		StatusCode: http.StatusServiceUnavailable,
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	zhtest.Serve(handler, req)

	// Test JSON response
	req = zhtest.NewRequest(http.MethodGet, "/test").WithHeader("Accept", "application/json").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		IsProblemDetail().
		ProblemDetailDetail("Too many requests, please slow down").
		HeaderExists("Retry-After")

	// Test plain text response
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w = zhtest.Serve(handler, req)
	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
}

func TestIPKeyExtractor(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("X-Forwarded-For", "192.168.1.1").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	key := IPKeyExtractor()(req)
	if key != "192.168.1.1" {
		t.Errorf("expected key '192.168.1.1', got '%s'", key)
	}
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	key = IPKeyExtractor()(req)
	if key != "127.0.0.1" {
		t.Errorf("expected key '127.0.0.1', got '%s'", key)
	}
}

func TestRateLimit_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := RateLimit(config.RateLimitConfig{
		Rate:      1,
		Window:    time.Second,
		Algorithm: config.TokenBucket,
	})

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})
	wrapped := metricsMw(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// First request allowed
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	w1 := httptest.NewRecorder()
	wrapped.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w1.Code)
	}

	// Second request rejected
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	w2 := httptest.NewRecorder()
	wrapped.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w2.Code)
	}

	// Check metrics
	families := reg.Gather()
	var counter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "ratelimit_allowed_total" || f.Name == "ratelimit_rejected_total" {
			counter = &f
			break
		}
	}
	if counter == nil {
		t.Fatal("expected ratelimit metrics")
	}
}

func TestHeaderKeyExtractor(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
		expected    string
	}{
		{
			name:        "X-API-Key header present",
			headerName:  "X-API-Key",
			headerValue: "api-key-123",
			expected:    "api-key-123",
		},
		{
			name:        "header missing",
			headerName:  "X-User-ID",
			headerValue: "",
			expected:    "",
		},
		{
			name:        "empty header value",
			headerName:  "X-Auth",
			headerValue: "",
			expected:    "",
		},
		{
			name:        "custom header",
			headerName:  "X-Custom-Key",
			headerValue: "custom-value",
			expected:    "custom-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := HeaderKeyExtractor(tt.headerName)
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			if tt.headerValue != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}

			result := extractor(req)
			if result != tt.expected {
				t.Errorf("expected key '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestContextKeyExtractor(t *testing.T) {
	tests := []struct {
		name         string
		contextKey   string
		contextValue any
		expected     string
	}{
		{
			name:         "string value in context",
			contextKey:   "user_id",
			contextValue: "user123",
			expected:     "user123",
		},
		{
			name:         "integer value in context",
			contextKey:   "user_id",
			contextValue: 456,
			expected:     "",
		},
		{
			name:         "missing key",
			contextKey:   "user_id",
			contextValue: nil,
			expected:     "",
		},
		{
			name:         "different key",
			contextKey:   "tenant_id",
			contextValue: "tenant-abc",
			expected:     "tenant-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := ContextKeyExtractor(tt.contextKey)
			ctx := context.Background()
			if tt.contextValue != nil {
				//nolint:staticcheck // Using string key is acceptable in tests
				ctx = context.WithValue(ctx, tt.contextKey, tt.contextValue)
			}
			req := zhtest.NewRequest(http.MethodGet, "/test").Build().WithContext(ctx)

			result := extractor(req)
			if result != tt.expected {
				t.Errorf("expected key '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestJWTSubjectKeyExtractor(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected string
	}{
		{
			name:     "has subject claim",
			claims:   HS256Claims{"sub": "user123"},
			expected: "user123",
		},
		{
			name:     "missing subject claim",
			claims:   HS256Claims{"iss": "my-issuer"},
			expected: "",
		},
		{
			name:     "empty claims",
			claims:   HS256Claims{},
			expected: "",
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: "",
		},
		{
			name:     "subject as map",
			claims:   map[string]any{"sub": "map-user"},
			expected: "map-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := JWTSubjectKeyExtractor()
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := zhtest.NewRequest(http.MethodGet, "/test").Build().WithContext(ctx)

			result := extractor(req)
			if result != tt.expected {
				t.Errorf("expected key '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCompositeKeyExtractor(t *testing.T) {
	tests := []struct {
		name       string
		extractors []config.KeyExtractor
		setupReq   func(*http.Request)
		expected   string
	}{
		{
			name: "first extractor returns value",
			extractors: []config.KeyExtractor{
				func(r *http.Request) string { return "first" },
				func(r *http.Request) string { return "second" },
			},
			setupReq: func(r *http.Request) {},
			expected: "first",
		},
		{
			name: "first empty, second returns value",
			extractors: []config.KeyExtractor{
				func(r *http.Request) string { return "" },
				func(r *http.Request) string { return "second" },
			},
			setupReq: func(r *http.Request) {},
			expected: "second",
		},
		{
			name: "all empty",
			extractors: []config.KeyExtractor{
				func(r *http.Request) string { return "" },
				func(r *http.Request) string { return "" },
			},
			setupReq: func(r *http.Request) {},
			expected: "",
		},
		{
			name:       "empty extractors list",
			extractors: []config.KeyExtractor{},
			setupReq:   func(r *http.Request) {},
			expected:   "",
		},
		{
			name: "realistic: JWT subject then header then IP",
			extractors: []config.KeyExtractor{
				JWTSubjectKeyExtractor(),
				HeaderKeyExtractor("X-API-Key"),
				IPKeyExtractor(),
			},
			setupReq: func(r *http.Request) {
				r.Header.Set("X-API-Key", "api-123")
				r.RemoteAddr = "192.168.1.1:8080"
			},
			expected: "api-123",
		},
		{
			name: "realistic: JWT empty, falls back to IP",
			extractors: []config.KeyExtractor{
				JWTSubjectKeyExtractor(),
				IPKeyExtractor(),
			},
			setupReq: func(r *http.Request) {
				r.RemoteAddr = "10.0.0.1:9090"
			},
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := CompositeKeyExtractor(tt.extractors...)
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			tt.setupReq(req)

			result := extractor(req)
			if result != tt.expected {
				t.Errorf("expected key '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIPKeyExtractor_IPv6(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		expected      string
	}{
		{
			name:       "IPv6 localhost",
			remoteAddr: "[::1]:8080",
			expected:   "::1",
		},
		{
			name:       "IPv6 full address",
			remoteAddr: "[2001:db8::1]:443",
			expected:   "2001:db8::1",
		},
		{
			name:          "IPv6 with forwarded",
			remoteAddr:    "[::1]:8080",
			xForwardedFor: "2001:db8::42",
			expected:      "2001:db8::42",
		},
		{
			name:       "no port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
		{
			name:          "forwarded with multiple IPs",
			remoteAddr:    "10.0.0.1:1234",
			xForwardedFor: "203.0.113.1, 198.51.100.1, 192.168.1.1",
			expected:      "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set(httpx.HeaderXForwardedFor, tt.xForwardedFor)
			}

			extractor := IPKeyExtractor()
			result := extractor(req)
			if result != tt.expected {
				t.Errorf("expected key '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestRateLimit_IncludedPaths(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:          1,
		Window:        time.Second,
		Algorithm:     config.TokenBucket,
		IncludedPaths: []string{"/api/", "/admin"},
	})
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"allowed path - first request", "/api/users", http.StatusOK},
		{"allowed path - rate limited", "/api/users", http.StatusTooManyRequests},
		// Note: /admin shares the same rate limit (same IP), so it's also rate limited
		{"allowed exact path - also rate limited", "/admin", http.StatusTooManyRequests},
		{"non-allowed path - not rate limited 1", "/health", http.StatusOK},
		{"non-allowed path - not rate limited 2", "/health", http.StatusOK},
		{"non-allowed path - not rate limited 3", "/metrics", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := zhtest.NewRequest(http.MethodGet, tt.path).Build()
			req.RemoteAddr = "127.0.0.1:12345"
			w := zhtest.Serve(handler, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestRateLimit_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = RateLimit(config.RateLimitConfig{
		Rate:          1,
		Window:        time.Second,
		Algorithm:     config.TokenBucket,
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api"},
	})
}

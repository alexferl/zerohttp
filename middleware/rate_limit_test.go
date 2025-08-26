package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

func TestRateLimit_DefaultConfigFallbacks(t *testing.T) {
	tests := []struct {
		name   string
		option func() func(http.Handler) http.Handler
	}{
		{"rate", func() func(http.Handler) http.Handler { return RateLimit(config.WithRateLimitRate(0)) }},
		{"window", func() func(http.Handler) http.Handler { return RateLimit(config.WithRateLimitWindow(0)) }},
		{"algorithm", func() func(http.Handler) http.Handler { return RateLimit(config.WithRateLimitAlgorithm("")) }},
		{"key extractor", func() func(http.Handler) http.Handler { return RateLimit(config.WithRateLimitKeyExtractor(nil)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.option()
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestRateLimitStatusCodeAndMessageDefaults(t *testing.T) {
	m := RateLimit(config.WithRateLimitStatusCode(0), config.WithRateLimitRate(1), config.WithRateLimitWindow(time.Minute))
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "Rate limit exceeded" {
		t.Errorf("expected default message, got %q", body)
	}
}

func TestRateLimitMessageDefaults(t *testing.T) {
	m := RateLimit(config.WithRateLimitMessage(""), config.WithRateLimitRate(1), config.WithRateLimitWindow(time.Minute))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := m(next)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req) // 1st OK
	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req) // 2nd rate limited

	if body := rr.Body.String(); body != "Rate limit exceeded" {
		t.Errorf("expected default message, got %q", body)
	}
}

func TestRateLimitTokenBucket(t *testing.T) {
	middleware := RateLimit(
		config.WithRateLimitRate(2),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
	)
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	for i := range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}
	if count != 2 {
		t.Errorf("expected 2 successful requests, got %d", count)
	}
	if body := rr.Body.String(); body != "Rate limit exceeded" {
		t.Errorf("expected rate limit message, got %s", body)
	}
}

func TestRateLimitFixedWindow(t *testing.T) {
	middleware := RateLimit(config.WithRateLimitRate(3), config.WithRateLimitWindow(time.Second), config.WithRateLimitAlgorithm(config.FixedWindow))
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	for i := range 3 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}
	if count != 3 {
		t.Errorf("expected 3 successful requests, got %d", count)
	}
}

func TestRateLimitSlidingWindow(t *testing.T) {
	m := RateLimit(config.WithRateLimitRate(2), config.WithRateLimitWindow(100*time.Millisecond), config.WithRateLimitAlgorithm(config.SlidingWindow))
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	for i := range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rr.Code)
	}

	time.Sleep(110 * time.Millisecond)
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 after window slide, got %d", rr.Code)
	}
}

func TestRateLimitHeaders(t *testing.T) {
	middleware := RateLimit(
		config.WithRateLimitRate(5),
		config.WithRateLimitWindow(time.Minute),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitIncludeHeaders(true),
	)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("expected X-RateLimit-Limit '5', got '%s'", rr.Header().Get("X-RateLimit-Limit"))
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header to be set")
	}
	if rr.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header to be set")
	}
	if rr.Header().Get("X-RateLimit-Window") != "1m0s" {
		t.Errorf("expected X-RateLimit-Window '1m0s', got '%s'", rr.Header().Get("X-RateLimit-Window"))
	}
}

func TestRateLimitNoHeaders(t *testing.T) {
	middleware := RateLimit(
		config.WithRateLimitRate(2),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitIncludeHeaders(false),
	)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	for _, hdr := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if rr.Header().Get(hdr) != "" {
			t.Errorf("expected no %s header, got '%s'", hdr, rr.Header().Get(hdr))
		}
	}
}

func TestRateLimitCustomKeyExtractor(t *testing.T) {
	middleware := RateLimit(
		config.WithRateLimitRate(2),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitKeyExtractor(func(r *http.Request) string {
			return r.Header.Get("User-ID")
		}),
	)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	// User 1 allowed first two
	for i := range 2 {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-ID", "user1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("user1 request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-ID", "user1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("user1: expected status 429, got %d", rr.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-ID", "user2")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("user2: expected status 200, got %d", rr.Code)
	}
}

func TestRateLimitExemptPaths(t *testing.T) {
	middleware := RateLimit(
		config.WithRateLimitRate(1),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitExemptPaths([]string{"/health", "/metrics"}),
	)
	count := 0
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	for i := range 3 {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("exempt path request %d: expected status 200, got %d", i+1, rr.Code)
		}
	}
	if count != 3 {
		t.Errorf("expected 3 requests to exempt path, got %d", count)
	}
	req := httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("first non-exempt request: expected status 200, got %d", rr.Code)
	}

	req = httptest.NewRequest("GET", "/api", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("second non-exempt request: expected status 429, got %d", rr.Code)
	}
}

func TestRateLimitCustomMessage(t *testing.T) {
	middleware := RateLimit(
		config.WithRateLimitRate(1),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitMessage("Too many requests, please slow down"),
		config.WithRateLimitStatusCode(http.StatusServiceUnavailable),
	)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("second request: expected status 503, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "Too many requests, please slow down" {
		t.Errorf("expected custom message, got %s", body)
	}
	if retryAfter := rr.Header().Get("Retry-After"); retryAfter == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestRateLimitDefaultKeyExtractor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	req.RemoteAddr = "127.0.0.1:12345"
	key := config.DefaultKeyExtractor(req)
	if key != "192.168.1.1" {
		t.Errorf("expected key '192.168.1.1', got '%s'", key)
	}
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	key = config.DefaultKeyExtractor(req)
	if key != "127.0.0.1:12345" {
		t.Errorf("expected key '127.0.0.1:12345', got '%s'", key)
	}
}

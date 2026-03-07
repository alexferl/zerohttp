package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
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
			req := zhtest.NewRequest(http.MethodGet, "/test").Build()
			w := zhtest.TestMiddleware(m, req)

			zhtest.AssertWith(t, w).Status(http.StatusOK)
		})
	}
}

func TestRateLimitStatusCodeAndMessageDefaults(t *testing.T) {
	m := RateLimit(config.WithRateLimitStatusCode(0), config.WithRateLimitRate(1), config.WithRateLimitWindow(time.Minute))
	handler := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(handler, req)
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).Body("Rate limit exceeded")
}

func TestRateLimitMessageDefaults(t *testing.T) {
	m := RateLimit(config.WithRateLimitMessage(""), config.WithRateLimitRate(1), config.WithRateLimitWindow(time.Minute))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := m(next)
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(handler, req) // 1st OK
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(handler, req) // 2nd rate limited

	zhtest.AssertWith(t, w).Body("Rate limit exceeded")
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

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).Body("Rate limit exceeded")
	if count != 2 {
		t.Errorf("expected 2 successful requests, got %d", count)
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
	m := RateLimit(config.WithRateLimitRate(2), config.WithRateLimitWindow(100*time.Millisecond), config.WithRateLimitAlgorithm(config.SlidingWindow))
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
	middleware := RateLimit(
		config.WithRateLimitRate(5),
		config.WithRateLimitWindow(time.Minute),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitIncludeHeaders(true),
	)
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
	middleware := RateLimit(
		config.WithRateLimitRate(2),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitIncludeHeaders(false),
	)
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
		req := zhtest.NewRequest(http.MethodGet, "/health").Build()
		req.RemoteAddr = "127.0.0.1:12345"
		w := zhtest.Serve(handler, req)

		if w.Code != http.StatusOK {
			t.Errorf("exempt path request %d: expected status 200, got %d", i+1, w.Code)
		}
	}
	if count != 3 {
		t.Errorf("expected 3 requests to exempt path, got %d", count)
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
	middleware := RateLimit(
		config.WithRateLimitRate(1),
		config.WithRateLimitWindow(time.Second),
		config.WithRateLimitAlgorithm(config.TokenBucket),
		config.WithRateLimitMessage("Too many requests, please slow down"),
		config.WithRateLimitStatusCode(http.StatusServiceUnavailable),
	)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	zhtest.Serve(handler, req)

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		Body("Too many requests, please slow down").
		HeaderExists("Retry-After")
}

func TestRateLimitDefaultKeyExtractor(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("X-Forwarded-For", "192.168.1.1").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	key := config.DefaultKeyExtractor(req)
	if key != "192.168.1.1" {
		t.Errorf("expected key '192.168.1.1', got '%s'", key)
	}
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	key = config.DefaultKeyExtractor(req)
	if key != "127.0.0.1:12345" {
		t.Errorf("expected key '127.0.0.1:12345', got '%s'", key)
	}
}

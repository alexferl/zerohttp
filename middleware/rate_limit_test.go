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
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).IsProblemDetail().ProblemDetailDetail("Rate limit exceeded")
}

func TestRateLimitMessageDefaults(t *testing.T) {
	m := RateLimit(config.RateLimitConfig{Message: "", Rate: 1, Window: time.Minute})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := m(next)
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(handler, req) // 1st OK
	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(handler, req) // 2nd rate limited

	zhtest.AssertWith(t, w).IsProblemDetail().ProblemDetailDetail("Rate limit exceeded")
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
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusTooManyRequests).IsProblemDetail().ProblemDetailDetail("Rate limit exceeded")
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

func TestRateLimitExemptPaths(t *testing.T) {
	middleware := RateLimit(config.RateLimitConfig{
		Rate:        1,
		Window:      time.Second,
		Algorithm:   config.TokenBucket,
		ExemptPaths: []string{"/health", "/metrics"},
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

	req = zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "127.0.0.1:12345"
	w := zhtest.Serve(handler, req)

	zhtest.AssertWith(t, w).
		Status(http.StatusServiceUnavailable).
		IsProblemDetail().
		ProblemDetailDetail("Too many requests, please slow down").
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

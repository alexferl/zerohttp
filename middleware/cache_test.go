package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestCache_Basic(t *testing.T) {
	t.Run("caches GET responses", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response " + r.URL.Query().Get("id")))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		// First request - should hit handler
		req1 := httptest.NewRequest(http.MethodGet, "/test?id=1", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)
		zhtest.AssertWith(t, w1).Status(http.StatusOK).Body("response 1")
		if callCount != 1 {
			t.Errorf("Expected 1 handler call, got %d", callCount)
		}

		// Second request - should be cached
		req2 := httptest.NewRequest(http.MethodGet, "/test?id=1", nil)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)
		zhtest.AssertWith(t, w2).Status(http.StatusOK).Body("response 1")
		if callCount != 1 {
			t.Errorf("Expected still 1 handler call, got %d (should be cached)", callCount)
		}

		// Different query - should hit handler
		req3 := httptest.NewRequest(http.MethodGet, "/test?id=2", nil)
		w3 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w3, req3)
		zhtest.AssertWith(t, w3).Status(http.StatusOK).Body("response 2")
		if callCount != 2 {
			t.Errorf("Expected 2 handler calls, got %d", callCount)
		}
	})

	t.Run("does not cache POST requests", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("post response"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
		})

		req1 := httptest.NewRequest(http.MethodPost, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		req2 := httptest.NewRequest(http.MethodPost, "/test", nil)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls for POST, got %d", callCount)
		}
	})

	t.Run("respects no-cache header", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
		})

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		// Second request with no-cache
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set("Cache-Control", "no-cache")
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls with no-cache, got %d", callCount)
		}
	})

	t.Run("respects exempt paths", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:  time.Minute,
			ExemptPaths: []string{"/api/live*"},
		})

		req1 := httptest.NewRequest(http.MethodGet, "/api/live", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		req2 := httptest.NewRequest(http.MethodGet, "/api/live", nil)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls for exempt path, got %d", callCount)
		}
	})
}

func TestCache_ConditionalRequests(t *testing.T) {
	t.Run("returns 304 on ETag match", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
			ETag:       true,
		})

		// First request to cache
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		etag := w1.Header().Get("ETag")
		if etag == "" {
			t.Fatal("Expected ETag to be set")
		}

		// Second request with If-None-Match
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set("If-None-Match", etag)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertWith(t, w2).Status(http.StatusNotModified)
		if w2.Body.String() != "" {
			t.Errorf("Expected empty body for 304, got %q", w2.Body.String())
		}
	})

	t.Run("returns 304 on Last-Modified match", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			LastModified: true,
		})

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		lastModified := w1.Header().Get("Last-Modified")
		if lastModified == "" {
			t.Fatal("Expected Last-Modified to be set")
		}

		// Second request with If-Modified-Since
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set("If-Modified-Since", lastModified)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertWith(t, w2).Status(http.StatusNotModified)
	})
}

func TestCache_VaryHeaders(t *testing.T) {
	t.Run("caches separately for different Accept headers", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("Content-Type", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
			Vary:       []string{"Accept"},
		})

		// JSON request
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1.Header.Set("Accept", "application/json")
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		// XML request - should hit handler again
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2.Header.Set("Accept", "application/xml")
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls for different Accept headers, got %d", callCount)
		}

		// Same JSON request - should be cached
		req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req3.Header.Set("Accept", "application/json")
		w3 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w3, req3)

		if callCount != 2 {
			t.Errorf("Expected still 2 calls, got %d", callCount)
		}
	})
}

func TestCache_HEAD(t *testing.T) {
	t.Run("HEAD returns cached headers without body", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "value")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
		})

		// GET to populate cache
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		// HEAD should return headers without body
		req2 := httptest.NewRequest(http.MethodHead, "/test", nil)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertWith(t, w2).Status(http.StatusOK)
		if w2.Body.String() != "" {
			t.Errorf("HEAD should have empty body, got %q", w2.Body.String())
		}
		if w2.Header().Get("X-Custom") != "value" {
			t.Error("HEAD should have cached headers")
		}
	})
}

func TestCache_Metrics(t *testing.T) {
	t.Run("emits cache hit and miss metrics", func(t *testing.T) {
		reg := metrics.NewRegistry()
		ctx := metrics.WithRegistry(context.Background(), reg)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
		})

		// First request - cache miss
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req1 = req1.WithContext(ctx)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - cache hit
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req2 = req2.WithContext(ctx)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		// Third request - cache hit
		req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		req3 = req3.WithContext(ctx)
		w3 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w3, req3)

		// Check metrics
		families := reg.Gather()
		var counter *metrics.MetricFamily
		for _, f := range families {
			if f.Name == "cache_requests_total" {
				counter = &f
				break
			}
		}
		if counter == nil {
			t.Fatal("expected cache_requests_total metric")
		}

		results := make(map[string]uint64)
		for _, m := range counter.Metrics {
			results[m.Labels["result"]] = m.Counter
		}

		if results["miss"] != 1 {
			t.Errorf("expected 1 miss, got %d", results["miss"])
		}
		if results["hit"] != 2 {
			t.Errorf("expected 2 hits, got %d", results["hit"])
		}
	})
}

func TestCache_NonCacheableResponseBody(t *testing.T) {
	t.Run("does not drop body for non-cacheable status codes", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found body"))
		})

		// Only cache 200 OK, not 404
		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:  time.Minute,
			StatusCodes: []int{http.StatusOK},
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound).Body("not found body")
	})

	t.Run("does not drop body for 500 errors", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("error message"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL: time.Minute,
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusInternalServerError).Body("error message")
	})
}

func TestCache_NoDuplicateHeaders(t *testing.T) {
	t.Run("does not duplicate handler-set headers", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Custom", "value")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"test": true}`))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		// Check that headers appear exactly once
		contentTypeValues := w.Header()["Content-Type"]
		if len(contentTypeValues) != 1 {
			t.Errorf("expected Content-Type to appear exactly once, got %d times: %v", len(contentTypeValues), contentTypeValues)
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type to be application/json, got %s", w.Header().Get("Content-Type"))
		}

		xCustomValues := w.Header()["X-Custom"]
		if len(xCustomValues) != 1 {
			t.Errorf("expected X-Custom to appear exactly once, got %d times: %v", len(xCustomValues), xCustomValues)
		}
	})

	t.Run("does not duplicate middleware-set headers on cache hit", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("X-Custom", "handler-value")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":"test"}`))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		// First request - simulate middleware setting headers before cache
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		w1.Header().Set("X-Security-Header", "security-value")
		w1.Header().Set("X-Request-Id", "req-123")

		cacheMiddleware(handler).ServeHTTP(w1, req1)

		if callCount != 1 {
			t.Errorf("expected 1 handler call, got %d", callCount)
		}

		// Second request - cache hit, simulate different request ID from middleware
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		w2.Header().Set("X-Security-Header", "security-value")
		w2.Header().Set("X-Request-Id", "req-456")

		cacheMiddleware(handler).ServeHTTP(w2, req2)

		if callCount != 1 {
			t.Errorf("expected still 1 handler call (cached), got %d", callCount)
		}

		// Verify no duplicate security headers
		securityHeaders := w2.Header()["X-Security-Header"]
		if len(securityHeaders) != 1 {
			t.Errorf("X-Security-Header should appear exactly once, got %d: %v", len(securityHeaders), securityHeaders)
		}

		// Request ID should be the NEW one (from middleware), not cached
		requestIDs := w2.Header()["X-Request-Id"]
		if len(requestIDs) != 1 {
			t.Errorf("X-Request-Id should appear exactly once, got %d: %v", len(requestIDs), requestIDs)
		}
		if w2.Header().Get("X-Request-Id") != "req-456" {
			t.Errorf("X-Request-Id should be 'req-456' (from middleware), got %q", w2.Header().Get("X-Request-Id"))
		}

		// Handler's custom header should be present from cache
		if w2.Header().Get("X-Custom") != "handler-value" {
			t.Errorf("X-Custom should be 'handler-value' from cache, got %q", w2.Header().Get("X-Custom"))
		}
	})

	t.Run("preserves multi-value headers from cache", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Add("X-Multi", "value1")
			w.Header().Add("X-Multi", "value2")
			w.Header().Add("X-Multi", "value3")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":"test"}`))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - cache hit
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		// All three values should be present
		multiHeaders := w2.Header()["X-Multi"]
		if len(multiHeaders) != 3 {
			t.Errorf("X-Multi should have 3 values, got %d: %v", len(multiHeaders), multiHeaders)
		}
	})
}

func TestCache_Flush(t *testing.T) {
	t.Run("flush switches to pass-through mode", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("before flush"))

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			_, _ = w.Write([]byte("after flush"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		// ResponseRecorder doesn't support real flushing, but we should still get the full body
		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("before flushafter flush")

		// Content-Type should be set
		if w.Header().Get("Content-Type") != "text/plain" {
			t.Errorf("expected Content-Type to be text/plain, got %s", w.Header().Get("Content-Type"))
		}
	})

	t.Run("flush does not cache response", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("response " + fmt.Sprintf("%d", callCount)))

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		// First request
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should hit handler again since flushed responses aren't cached
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w2, req2)

		if callCount != 2 {
			t.Errorf("expected 2 handler calls (flushed response not cached), got %d", callCount)
		}
	})

	t.Run("flush preserves non-200 status code", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("partial"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			_, _ = w.Write([]byte(" content"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusPartialContent).Body("partial content")
	})

	t.Run("non-cacheable response with flush does not double WriteHeader", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		})

		// Only cache 200 OK, so 404 is non-cacheable
		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			StatusCodes:  []int{http.StatusOK},
			CacheControl: "public, max-age=60",
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusNotFound).Body("not found")
	})

	t.Run("flush before WriteHeader does not double-write status", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush() // flush before WriteHeader
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			CacheControl: "public, max-age=60",
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusOK).Body("data")
	})
}

func TestCache_BodyOverflow(t *testing.T) {
	t.Run("preserves non-200 status when body overflows maxBodySize", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusPartialContent)
			// Write more than maxBodySize (10 bytes)
			_, _ = w.Write([]byte("this is a long response that exceeds ten bytes"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:   time.Minute,
			MaxBodySize:  10, // Very small to force overflow
			StatusCodes:  []int{http.StatusOK, http.StatusPartialContent},
			CacheControl: "public, max-age=60",
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		// Status should be 206, not 200
		zhtest.AssertWith(t, w).Status(http.StatusPartialContent)
		// Body should be complete
		if w.Body.String() != "this is a long response that exceeds ten bytes" {
			t.Errorf("unexpected body: %s", w.Body.String())
		}
	})

	t.Run("flush after body overflow does not double WriteHeader", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("exceeds ten bytes easily"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			_, _ = w.Write([]byte(" more data"))
		})

		cacheMiddleware := Cache(config.CacheConfig{
			DefaultTTL:  time.Minute,
			MaxBodySize: 10,
			StatusCodes: []int{http.StatusOK, http.StatusPartialContent},
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		cacheMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertWith(t, w).Status(http.StatusPartialContent)
	})
}

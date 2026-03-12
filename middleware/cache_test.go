package middleware

import (
	"context"
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

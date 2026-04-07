package idempotency

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestIdempotency_Basic(t *testing.T) {
	t.Run("caches POST responses and replays", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("X-Custom", "value")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request - should hit handler
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-123")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		zhtest.AssertWith(t, w1).Status(http.StatusCreated).Body(`{"id":"123"}`)
		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEmpty(t, w1.Header().Get(httpx.HeaderXIdempotencyReplay))

		// Second request with same key - should be cached
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-123")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertWith(t, w2).Status(http.StatusCreated).Body(`{"id":"123"}`)
		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, "true", w2.Header().Get(httpx.HeaderXIdempotencyReplay))
		zhtest.AssertEqual(t, "value", w2.Header().Get("X-Custom"))
	})

	t.Run("different body creates different cache entry", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + string(rune('0'+callCount)) + `"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-456")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Same key but different body - should hit handler again
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":200}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-456")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 2, callCount)
	})

	t.Run("does not cache error responses", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request - error
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-789")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should hit handler again (errors not cached)
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-789")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 2, callCount)
	})
}

func TestIdempotency_Methods(t *testing.T) {
	t.Run("only applies to state-changing methods", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// GET request should not be cached
		req1 := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-get")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		req2 := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-get")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 2, callCount)
	})

	t.Run("applies to PUT, PATCH, DELETE", func(t *testing.T) {
		methods := []string{http.MethodPut, http.MethodPatch, http.MethodDelete}

		for _, method := range methods {
			callCount := 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.WriteHeader(http.StatusOK)
			})

			idempotencyMiddleware := New(Config{
				TTL: time.Hour,
			})

			// First request
			req1 := httptest.NewRequest(method, "/api/resource", bytes.NewReader([]byte(`{}`)))
			req1.Header.Set(httpx.HeaderIdempotencyKey, "key-"+method)
			w1 := httptest.NewRecorder()
			idempotencyMiddleware(handler).ServeHTTP(w1, req1)

			// Second request - should be cached
			req2 := httptest.NewRequest(method, "/api/resource", bytes.NewReader([]byte(`{}`)))
			req2.Header.Set(httpx.HeaderIdempotencyKey, "key-"+method)
			w2 := httptest.NewRecorder()
			idempotencyMiddleware(handler).ServeHTTP(w2, req2)

			zhtest.AssertEqual(t, 1, callCount)
		}
	})
}

func TestIdempotency_Required(t *testing.T) {
	t.Run("returns 400 when key is required but missing", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL:      time.Hour,
			Required: true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertEqual(t, http.StatusBadRequest, w.Code)

		// Test JSON response
		req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
		w = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)
		zhtest.AssertWith(t, w).IsProblemDetail().ProblemDetailDetail("Idempotency-Key header is required")

		// Test JSON response (defaults to JSON without Accept header)
		req = httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{}`)))
		w = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)
		zhtest.AssertWith(t, w).Header(httpx.HeaderContentType, "application/problem+json")
	})

	t.Run("allows request when key is provided and required", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})

		idempotencyMiddleware := New(Config{
			TTL:      time.Hour,
			Required: true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-required")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertEqual(t, http.StatusCreated, w.Code)
	})
}

func TestIdempotency_ExcludedPaths(t *testing.T) {
	t.Run("skips idempotency for excluded paths", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL:           time.Hour,
			ExcludedPaths: []string{"/webhook*"},
		})

		// First request to excluded path
		req1 := httptest.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewReader([]byte(`{}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-webhook")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request to excluded path
		req2 := httptest.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewReader([]byte(`{}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-webhook")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 2, callCount)
	})
}

func TestIdempotency_MaxBodySize(t *testing.T) {
	t.Run("skips caching for bodies exceeding max size", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL:         time.Hour,
			MaxBodySize: 10, // Very small
		})

		// First request with large body
		req1 := httptest.NewRequest(http.MethodPost, "/api/upload", bytes.NewReader([]byte(`{"data":"this is a large body"}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-large")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should hit handler again
		req2 := httptest.NewRequest(http.MethodPost, "/api/upload", bytes.NewReader([]byte(`{"data":"this is a large body"}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-large")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 2, callCount)
	})
}

func TestIdempotency_CustomHeaderName(t *testing.T) {
	t.Run("uses custom header name", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL:        time.Hour,
			HeaderName: "X-Idempotency-Key",
		})

		// First request with custom header
		req1 := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader([]byte(`{}`)))
		req1.Header.Set("X-Idempotency-Key", "custom-key")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request with same custom header - should be cached
		req2 := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader([]byte(`{}`)))
		req2.Header.Set("X-Idempotency-Key", "custom-key")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, "true", w2.Header().Get(httpx.HeaderXIdempotencyReplay))
	})
}

func TestIdempotency_BodyPreservation(t *testing.T) {
	t.Run("request body is preserved for handler", func(t *testing.T) {
		var receivedBody []byte
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		body := []byte(`{"amount":100,"currency":"USD"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader(body))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-body")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertTrue(t, bytes.Equal(receivedBody, body))
	})
}

func TestIdempotency_ConcurrentLock(t *testing.T) {
	t.Run("concurrent requests with same key wait for lock holder", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			time.Sleep(50 * time.Millisecond) // Simulate slow operation
			w.Header().Set("X-Handler", "first")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL:               time.Hour,
			LockRetryInterval: 5 * time.Millisecond,
			LockMaxRetries:    100,
		})

		// Start first request (will acquire lock)
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "concurrent-key")
		w1 := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			idempotencyMiddleware(handler).ServeHTTP(w1, req1)
			close(done)
		}()

		// Small delay to ensure first request acquires lock
		time.Sleep(10 * time.Millisecond)

		// Start second request (will wait for lock)
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "concurrent-key")
		w2 := httptest.NewRecorder()

		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		// Wait for first request to complete
		<-done

		zhtest.AssertEqual(t, 1, callCount)

		// Second request should get replayed response
		zhtest.AssertEqual(t, http.StatusCreated, w2.Code)
		zhtest.AssertEqual(t, "true", w2.Header().Get(httpx.HeaderXIdempotencyReplay))
		zhtest.AssertEqual(t, "first", w2.Header().Get("X-Handler"))
	})

	t.Run("returns 409 when lock retries exhausted", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond) // Very slow operation
			w.WriteHeader(http.StatusCreated)
		})

		idempotencyMiddleware := New(Config{
			TTL:               time.Hour,
			LockRetryInterval: 5 * time.Millisecond,
			LockMaxRetries:    5, // Very short retry limit
		})

		// Start first request (will acquire lock)
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "slow-key")
		w1 := httptest.NewRecorder()

		go idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Small delay to ensure first request acquires lock
		time.Sleep(10 * time.Millisecond)

		// Start second request (will exhaust retries)
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "slow-key")
		w2 := httptest.NewRecorder()

		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, http.StatusConflict, w2.Code)

		// Test JSON response
		req2.Header.Set("Accept", "application/json")
		w2 = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)
		zhtest.AssertWith(t, w2).IsProblemDetail().ProblemDetailDetail("Idempotent request is still being processed")

		// Test JSON response (defaults to JSON without Accept header)
		req2 = httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "slow-key")
		w2 = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)
		// Note: This may return 409 (still processing) or 201 (completed) depending on timing
		// The important thing is that when it returns 409, the content type is JSON
		if w2.Code == http.StatusConflict {
			zhtest.AssertWith(t, w2).Header(httpx.HeaderContentType, "application/problem+json")
		}
	})
}

func TestIdempotency_StoreLockUnlock(t *testing.T) {
	t.Run("lock prevents concurrent execution", func(t *testing.T) {
		store := NewMemoryStore(100)
		ctx := context.Background()

		// First lock should succeed
		locked, err := store.Lock(ctx, "test-key")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, locked)

		// Second lock should fail (already locked)
		locked2, err := store.Lock(ctx, "test-key")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, locked2)

		// Unlock should succeed
		err = store.Unlock(ctx, "test-key")
		zhtest.AssertNoError(t, err)

		// Lock should succeed again after unlock
		locked3, err := store.Lock(ctx, "test-key")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, locked3)

		// Cleanup
		_ = store.Unlock(ctx, "test-key")
	})

	t.Run("different keys can be locked independently", func(t *testing.T) {
		store := NewMemoryStore(100)
		ctx := context.Background()

		// Lock first key
		locked1, err := store.Lock(ctx, "key-1")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, locked1)

		// Lock second key should also succeed
		locked2, err := store.Lock(ctx, "key-2")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, locked2)

		// Cleanup
		_ = store.Unlock(ctx, "key-1")
		_ = store.Unlock(ctx, "key-2")
	})

	t.Run("unlock non-existent key is safe", func(t *testing.T) {
		store := NewMemoryStore(100)
		ctx := context.Background()

		// Unlocking a key that was never locked should not panic
		err := store.Unlock(ctx, "never-locked")
		zhtest.AssertNoError(t, err)
	})
}

func TestIdempotencyMemoryStore_MaxKeys(t *testing.T) {
	t.Run("removes expired entries when max keys reached", func(t *testing.T) {
		store := NewMemoryStore(2)
		ctx := context.Background()

		// Add two entries, one expired
		_ = store.Set(ctx, "key-1", Record{StatusCode: 200}, 1*time.Millisecond)
		_ = store.Set(ctx, "key-2", Record{StatusCode: 201}, time.Hour)

		// Wait for first entry to expire
		time.Sleep(2 * time.Millisecond)

		// Add third entry - should remove expired entry
		_ = store.Set(ctx, "key-3", Record{StatusCode: 202}, time.Hour)

		// key-1 should be gone (expired)
		_, found, _ := store.Get(ctx, "key-1")
		zhtest.AssertFalse(t, found)

		// key-2 and key-3 should exist
		_, found2, _ := store.Get(ctx, "key-2")
		zhtest.AssertTrue(t, found2)
		_, found3, _ := store.Get(ctx, "key-3")
		zhtest.AssertTrue(t, found3)
	})

	t.Run("removes oldest entry when max keys reached and none expired", func(t *testing.T) {
		store := NewMemoryStore(2)
		ctx := context.Background()

		// Add two entries with different expiries
		_ = store.Set(ctx, "key-1", Record{StatusCode: 200}, time.Hour)
		time.Sleep(1 * time.Millisecond) // Ensure different creation time
		_ = store.Set(ctx, "key-2", Record{StatusCode: 201}, time.Hour)

		// Add third entry - should remove oldest (key-1)
		_ = store.Set(ctx, "key-3", Record{StatusCode: 202}, time.Hour)

		// key-1 should be gone (oldest)
		_, found, _ := store.Get(ctx, "key-1")
		zhtest.AssertFalse(t, found)

		// key-2 and key-3 should exist
		_, found2, _ := store.Get(ctx, "key-2")
		zhtest.AssertTrue(t, found2)
		_, found3, _ := store.Get(ctx, "key-3")
		zhtest.AssertTrue(t, found3)
	})
}

func TestIdempotency_NoDuplicateHeaders(t *testing.T) {
	t.Run("does not duplicate headers already set by middleware", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set("X-Custom", "handler-value")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request - handler sets headers
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "dup-key")
		w1 := httptest.NewRecorder()

		// Simulate middleware that sets headers before idempotency
		w1.Header().Set("X-Security-Header", "security-value")
		w1.Header().Set(httpx.HeaderXRequestId, "req-123")

		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should replay cached response
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "dup-key")
		w2 := httptest.NewRecorder()

		// Simulate same middleware setting headers before idempotency replay
		w2.Header().Set("X-Security-Header", "security-value")
		w2.Header().Set(httpx.HeaderXRequestId, "req-456") // Different request ID

		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		// Verify no duplicate headers
		securityHeaders := w2.Header()["X-Security-Header"]
		zhtest.AssertEqual(t, 1, len(securityHeaders))

		requestIds := w2.Header()[httpx.HeaderXRequestId]
		zhtest.AssertEqual(t, 1, len(requestIds))
		// Should have the NEW request ID (from middleware), not the cached one
		zhtest.AssertEqual(t, "req-456", requestIds[0])

		// Custom header from handler should be present
		zhtest.AssertEqual(t, "handler-value", w2.Header().Get("X-Custom"))
		zhtest.AssertEqual(t, "true", w2.Header().Get(httpx.HeaderXIdempotencyReplay))
	})

	t.Run("preserves multi-value headers from cache", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Add("X-Multi", "value1")
			w.Header().Add("X-Multi", "value2")
			w.Header().Add("X-Multi", "value3")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "multi-key")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - replay
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "multi-key")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		// All three values should be present
		multiHeaders := w2.Header()["X-Multi"]
		zhtest.AssertEqual(t, 3, len(multiHeaders))
	})
}

func TestIdempotency_HandlerWritesNothing(t *testing.T) {
	t.Run("does not cache when handler writes nothing", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			// Handler returns without writing anything
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request - handler writes nothing
		req1 := httptest.NewRequest(http.MethodPost, "/api/empty", bytes.NewReader([]byte(`{}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "empty-key")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should hit handler again (nothing cached)
		req2 := httptest.NewRequest(http.MethodPost, "/api/empty", bytes.NewReader([]byte(`{}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "empty-key")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 2, callCount)
	})

	t.Run("caches when handler writes explicit 200", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":"test"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/data", bytes.NewReader([]byte(`{}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "data-key")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should be cached
		req2 := httptest.NewRequest(http.MethodPost, "/api/data", bytes.NewReader([]byte(`{}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "data-key")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, "true", w2.Header().Get(httpx.HeaderXIdempotencyReplay))
	})
}

// errorStore is a mock store that returns errors for testing error handling
type errorStore struct {
	failGet    bool
	failLock   bool
	failUnlock bool
	failSet    bool
}

func (e *errorStore) Get(ctx context.Context, key string) (Record, bool, error) {
	if e.failGet {
		return Record{}, false, errors.New("store get error")
	}
	return Record{}, false, nil
}

func (e *errorStore) Set(ctx context.Context, key string, record Record, ttl time.Duration) error {
	if e.failSet {
		return errors.New("store set error")
	}
	return nil
}

func (e *errorStore) Lock(ctx context.Context, key string) (bool, error) {
	if e.failLock {
		return false, errors.New("store lock error")
	}
	return true, nil
}

func (e *errorStore) Unlock(ctx context.Context, key string) error {
	if e.failUnlock {
		return errors.New("store unlock error")
	}
	return nil
}

func (e *errorStore) Close() error {
	return nil
}

func TestIdempotency_StoreErrors(t *testing.T) {
	t.Run("continues to handler when store Get fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failGet: true}
		idempotencyMiddleware := New(Config{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should fail open and call handler
		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, http.StatusCreated, w.Code)
	})

	t.Run("continues to handler when store Lock fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failLock: true}
		idempotencyMiddleware := New(Config{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-lock-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should fail open and call handler
		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, http.StatusCreated, w.Code)
	})

	t.Run("logs error but does not fail when store Unlock fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failUnlock: true}
		idempotencyMiddleware := New(Config{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-unlock-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should complete successfully even if unlock fails
		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, http.StatusCreated, w.Code)
	})

	t.Run("logs error but does not fail when store Set fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failSet: true}
		idempotencyMiddleware := New(Config{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-set-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should complete successfully even if set fails
		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, http.StatusCreated, w.Code)
	})
}

func TestIdempotency_PanicRecovery(t *testing.T) {
	t.Run("unlocks store when handler panics", func(t *testing.T) {
		store := NewMemoryStore(100)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("handler panic")
		})

		idempotencyMiddleware := New(Config{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-panic")
		w := httptest.NewRecorder()

		// Use defer/recover to catch the panic
		func() {
			defer func() {
				_ = recover() // Expected panic, ignore it
			}()
			idempotencyMiddleware(handler).ServeHTTP(w, req)
		}()

		// Verify lock was released by trying to lock again
		ctx := context.Background()
		locked, err := store.Lock(ctx, "key-panic:POST:/api/payments:fef5c3c40c3c0f3887720d0d0bc7e26d61ebd42d82697109469727e790f35837")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, locked)
	})
}

func TestIdempotency_WriteHeaderHopByHop(t *testing.T) {
	t.Run("skips hop-by-hop headers in cache", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.Header().Set(httpx.HeaderConnection, httpx.ConnectionKeepAlive)
			w.Header().Set(httpx.HeaderKeepAlive, "timeout=5")
			w.Header().Set("X-Custom", "value")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		// First request
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-hop")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - replay
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-hop")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		// Hop-by-hop headers should not be replayed
		zhtest.AssertEmpty(t, w2.Header().Get(httpx.HeaderConnection))
		zhtest.AssertEmpty(t, w2.Header().Get(httpx.HeaderKeepAlive))
		// Custom header should be replayed
		zhtest.AssertEqual(t, "value", w2.Header().Get("X-Custom"))
	})
}

func TestIdempotency_WriteHeaderIdempotent(t *testing.T) {
	t.Run("WriteHeader is idempotent", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			w.WriteHeader(http.StatusOK) // Second call should be ignored
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL: time.Hour,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-idempotent")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		zhtest.AssertEqual(t, http.StatusCreated, w.Code)
	})
}

func TestIdempotency_IncludedPaths(t *testing.T) {
	t.Run("only applies idempotency to included paths", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		idempotencyMiddleware := New(Config{
			TTL:           time.Hour,
			IncludedPaths: []string{"/api/payments", "/api/transfers/"},
		})

		// First request to allowed path
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-allowed")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request to same allowed path - should be cached
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-allowed")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 1, callCount)
		zhtest.AssertEqual(t, "true", w2.Header().Get(httpx.HeaderXIdempotencyReplay))

		// Request to non-allowed path - should not be cached
		req3 := httptest.NewRequest(http.MethodPost, "/api/other", bytes.NewReader([]byte(`{"data":"test"}`)))
		req3.Header.Set(httpx.HeaderIdempotencyKey, "key-other")
		w3 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w3, req3)

		// Second request to non-allowed path - should hit handler again
		req4 := httptest.NewRequest(http.MethodPost, "/api/other", bytes.NewReader([]byte(`{"data":"test"}`)))
		req4.Header.Set(httpx.HeaderIdempotencyKey, "key-other")
		w4 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w4, req4)

		zhtest.AssertEqual(t, 3, callCount)
	})

	t.Run("included paths with prefix match", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := New(Config{
			TTL:           time.Hour,
			IncludedPaths: []string{"/api/"},
		})

		// Request to path under prefix
		req1 := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader([]byte(`{}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-prefix")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		// Second request - should be cached
		req2 := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader([]byte(`{}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-prefix")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertEqual(t, 1, callCount)

		// Request outside prefix - should not be cached
		req3 := httptest.NewRequest(http.MethodPost, "/health", bytes.NewReader([]byte(`{}`)))
		req3.Header.Set(httpx.HeaderIdempotencyKey, "key-health")
		w3 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w3, req3)

		req4 := httptest.NewRequest(http.MethodPost, "/health", bytes.NewReader([]byte(`{}`)))
		req4.Header.Set(httpx.HeaderIdempotencyKey, "key-health")
		w4 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w4, req4)

		zhtest.AssertEqual(t, 3, callCount)
	})
}

func TestIdempotency_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			TTL:           time.Hour,
			ExcludedPaths: []string{"/webhook"},
			IncludedPaths: []string{"/api"},
		})
	})
}

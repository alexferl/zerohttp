package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL: time.Hour,
		})

		// First request - should hit handler
		req1 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req1.Header.Set(httpx.HeaderIdempotencyKey, "key-123")
		w1 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w1, req1)

		zhtest.AssertWith(t, w1).Status(http.StatusCreated).Body(`{"id":"123"}`)
		if callCount != 1 {
			t.Errorf("Expected 1 handler call, got %d", callCount)
		}
		if w1.Header().Get(httpx.HeaderXIdempotencyReplay) != "" {
			t.Error("First request should not have X-Idempotency-Replay header")
		}

		// Second request with same key - should be cached
		req2 := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "key-123")
		w2 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)

		zhtest.AssertWith(t, w2).Status(http.StatusCreated).Body(`{"id":"123"}`)
		if callCount != 1 {
			t.Errorf("Expected still 1 handler call, got %d (should be cached)", callCount)
		}
		if w2.Header().Get(httpx.HeaderXIdempotencyReplay) != "true" {
			t.Error("Replayed request should have X-Idempotency-Replay: true header")
		}
		if w2.Header().Get("X-Custom") != "value" {
			t.Error("Replayed response should have custom headers")
		}
	})

	t.Run("different body creates different cache entry", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + string(rune('0'+callCount)) + `"}`))
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls for different bodies, got %d", callCount)
		}
	})

	t.Run("does not cache error responses", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid"}`))
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls (errors not cached), got %d", callCount)
		}
	})
}

func TestIdempotency_Methods(t *testing.T) {
	t.Run("only applies to state-changing methods", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls for GET (not cached), got %d", callCount)
		}
	})

	t.Run("applies to PUT, PATCH, DELETE", func(t *testing.T) {
		methods := []string{http.MethodPut, http.MethodPatch, http.MethodDelete}

		for _, method := range methods {
			callCount := 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.WriteHeader(http.StatusOK)
			})

			idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

			if callCount != 1 {
				t.Errorf("Expected 1 handler call for %s (cached), got %d", method, callCount)
			}
		}
	})
}

func TestIdempotency_Required(t *testing.T) {
	t.Run("returns 400 when key is required but missing", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL:      time.Hour,
			Required: true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 when key is required, got %d", w.Code)
		}

		// Test JSON response
		req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
		w = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)
		zhtest.AssertWith(t, w).IsProblemDetail().ProblemDetailDetail("Idempotency-Key header is required")

		// Test plain text response
		req = httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{}`)))
		w = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)
		zhtest.AssertWith(t, w).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
	})

	t.Run("allows request when key is provided and required", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL:      time.Hour,
			Required: true,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-required")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201 when key is provided, got %d", w.Code)
		}
	})
}

func TestIdempotency_ExcludedPaths(t *testing.T) {
	t.Run("skips idempotency for excluded paths", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls for excluded path, got %d", callCount)
		}
	})
}

func TestIdempotency_MaxBodySize(t *testing.T) {
	t.Run("skips caching for bodies exceeding max size", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls (body too large), got %d", callCount)
		}
	})
}

func TestIdempotency_CustomHeaderName(t *testing.T) {
	t.Run("uses custom header name", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 1 {
			t.Errorf("Expected 1 handler call (cached), got %d", callCount)
		}
		if w2.Header().Get(httpx.HeaderXIdempotencyReplay) != "true" {
			t.Error("Expected X-Idempotency-Replay header on replay")
		}
	})
}

func TestIdempotency_BodyPreservation(t *testing.T) {
	t.Run("request body is preserved for handler", func(t *testing.T) {
		var receivedBody []byte
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL: time.Hour,
		})

		body := []byte(`{"amount":100,"currency":"USD"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader(body))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-body")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		if !bytes.Equal(receivedBody, body) {
			t.Errorf("Handler received different body. Expected %q, got %q", string(body), string(receivedBody))
		}
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 1 {
			t.Errorf("Expected 1 handler call for concurrent requests, got %d", callCount)
		}

		// Second request should get replayed response
		if w2.Code != http.StatusCreated {
			t.Errorf("Expected 201 for second request, got %d", w2.Code)
		}
		if w2.Header().Get(httpx.HeaderXIdempotencyReplay) != "true" {
			t.Error("Second request should have X-Idempotency-Replay: true header")
		}
		if w2.Header().Get("X-Handler") != "first" {
			t.Error("Second request should have headers from first handler response")
		}
	})

	t.Run("returns 409 when lock retries exhausted", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond) // Very slow operation
			w.WriteHeader(http.StatusCreated)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if w2.Code != http.StatusConflict {
			t.Errorf("Expected 409 Conflict, got %d", w2.Code)
		}

		// Test JSON response
		req2.Header.Set("Accept", "application/json")
		w2 = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)
		zhtest.AssertWith(t, w2).IsProblemDetail().ProblemDetailDetail("Idempotent request is still being processed")

		// Test plain text response
		req2 = httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req2.Header.Set(httpx.HeaderIdempotencyKey, "slow-key")
		w2 = httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w2, req2)
		// Note: This may return 409 (still processing) or 201 (completed) depending on timing
		// The important thing is that when it returns 409, the content type is plain text
		if w2.Code == http.StatusConflict {
			zhtest.AssertWith(t, w2).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
		}
	})
}

func TestIdempotency_StoreLockUnlock(t *testing.T) {
	t.Run("lock prevents concurrent execution", func(t *testing.T) {
		store := NewIdempotencyMemoryStore(100)
		ctx := context.Background()

		// First lock should succeed
		locked, err := store.Lock(ctx, "test-key")
		if err != nil {
			t.Errorf("Unexpected error on first lock: %v", err)
		}
		if !locked {
			t.Error("First lock should succeed")
		}

		// Second lock should fail (already locked)
		locked2, err := store.Lock(ctx, "test-key")
		if err != nil {
			t.Errorf("Unexpected error on second lock: %v", err)
		}
		if locked2 {
			t.Error("Second lock should fail when key is already locked")
		}

		// Unlock should succeed
		err = store.Unlock(ctx, "test-key")
		if err != nil {
			t.Errorf("Unexpected error on unlock: %v", err)
		}

		// Lock should succeed again after unlock
		locked3, err := store.Lock(ctx, "test-key")
		if err != nil {
			t.Errorf("Unexpected error on third lock: %v", err)
		}
		if !locked3 {
			t.Error("Lock should succeed after unlock")
		}

		// Cleanup
		_ = store.Unlock(ctx, "test-key")
	})

	t.Run("different keys can be locked independently", func(t *testing.T) {
		store := NewIdempotencyMemoryStore(100)
		ctx := context.Background()

		// Lock first key
		locked1, err := store.Lock(ctx, "key-1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !locked1 {
			t.Error("Lock for key-1 should succeed")
		}

		// Lock second key should also succeed
		locked2, err := store.Lock(ctx, "key-2")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !locked2 {
			t.Error("Lock for key-2 should succeed independently")
		}

		// Cleanup
		_ = store.Unlock(ctx, "key-1")
		_ = store.Unlock(ctx, "key-2")
	})

	t.Run("unlock non-existent key is safe", func(t *testing.T) {
		store := NewIdempotencyMemoryStore(100)
		ctx := context.Background()

		// Unlocking a key that was never locked should not panic
		err := store.Unlock(ctx, "never-locked")
		if err != nil {
			t.Errorf("Unexpected error unlocking non-existent key: %v", err)
		}
	})
}

func TestIdempotencyMemoryStore_MaxKeys(t *testing.T) {
	t.Run("removes expired entries when max keys reached", func(t *testing.T) {
		store := NewIdempotencyMemoryStore(2)
		ctx := context.Background()

		// Add two entries, one expired
		_ = store.Set(ctx, "key-1", config.IdempotencyRecord{StatusCode: 200}, 1*time.Millisecond)
		_ = store.Set(ctx, "key-2", config.IdempotencyRecord{StatusCode: 201}, time.Hour)

		// Wait for first entry to expire
		time.Sleep(2 * time.Millisecond)

		// Add third entry - should remove expired entry
		_ = store.Set(ctx, "key-3", config.IdempotencyRecord{StatusCode: 202}, time.Hour)

		// key-1 should be gone (expired)
		_, found, _ := store.Get(ctx, "key-1")
		if found {
			t.Error("key-1 should have been removed (expired)")
		}

		// key-2 and key-3 should exist
		_, found2, _ := store.Get(ctx, "key-2")
		if !found2 {
			t.Error("key-2 should exist")
		}
		_, found3, _ := store.Get(ctx, "key-3")
		if !found3 {
			t.Error("key-3 should exist")
		}
	})

	t.Run("removes oldest entry when max keys reached and none expired", func(t *testing.T) {
		store := NewIdempotencyMemoryStore(2)
		ctx := context.Background()

		// Add two entries with different expiries
		_ = store.Set(ctx, "key-1", config.IdempotencyRecord{StatusCode: 200}, time.Hour)
		time.Sleep(1 * time.Millisecond) // Ensure different creation time
		_ = store.Set(ctx, "key-2", config.IdempotencyRecord{StatusCode: 201}, time.Hour)

		// Add third entry - should remove oldest (key-1)
		_ = store.Set(ctx, "key-3", config.IdempotencyRecord{StatusCode: 202}, time.Hour)

		// key-1 should be gone (oldest)
		_, found, _ := store.Get(ctx, "key-1")
		if found {
			t.Error("key-1 should have been removed (oldest)")
		}

		// key-2 and key-3 should exist
		_, found2, _ := store.Get(ctx, "key-2")
		if !found2 {
			t.Error("key-2 should exist")
		}
		_, found3, _ := store.Get(ctx, "key-3")
		if !found3 {
			t.Error("key-3 should exist")
		}
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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
		if len(securityHeaders) != 1 {
			t.Errorf("X-Security-Header should appear once, got %d: %v", len(securityHeaders), securityHeaders)
		}

		requestIds := w2.Header()[httpx.HeaderXRequestId]
		if len(requestIds) != 1 {
			t.Errorf("X-Request-Id should appear once, got %d: %v", len(requestIds), requestIds)
		}
		// Should have the NEW request ID (from middleware), not the cached one
		if requestIds[0] != "req-456" {
			t.Errorf("X-Request-Id should be 'req-456' (from middleware), got %q", requestIds[0])
		}

		// Custom header from handler should be present
		if w2.Header().Get("X-Custom") != "handler-value" {
			t.Error("X-Custom header should be replayed from cache")
		}

		if w2.Header().Get(httpx.HeaderXIdempotencyReplay) != "true" {
			t.Error("Expected X-Idempotency-Replay header on replay")
		}
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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
		if len(multiHeaders) != 3 {
			t.Errorf("X-Multi should have 3 values, got %d: %v", len(multiHeaders), multiHeaders)
		}
	})
}

func TestIdempotency_HandlerWritesNothing(t *testing.T) {
	t.Run("does not cache when handler writes nothing", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			// Handler returns without writing anything
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 2 {
			t.Errorf("Expected 2 handler calls (nothing cached), got %d", callCount)
		}
	})

	t.Run("caches when handler writes explicit 200", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":"test"}`))
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 1 {
			t.Errorf("Expected 1 handler call (cached), got %d", callCount)
		}
		if w2.Header().Get(httpx.HeaderXIdempotencyReplay) != "true" {
			t.Error("Expected X-Idempotency-Replay header on replay")
		}
	})
}

// errorStore is a mock store that returns errors for testing error handling
type errorStore struct {
	failGet    bool
	failLock   bool
	failUnlock bool
	failSet    bool
}

func (e *errorStore) Get(ctx context.Context, key string) (config.IdempotencyRecord, bool, error) {
	if e.failGet {
		return config.IdempotencyRecord{}, false, errors.New("store get error")
	}
	return config.IdempotencyRecord{}, false, nil
}

func (e *errorStore) Set(ctx context.Context, key string, record config.IdempotencyRecord, ttl time.Duration) error {
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

func TestIdempotency_StoreErrors(t *testing.T) {
	t.Run("continues to handler when store Get fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failGet: true}
		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should fail open and call handler
		if callCount != 1 {
			t.Errorf("Expected 1 handler call (fail open), got %d", callCount)
		}
		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}
	})

	t.Run("continues to handler when store Lock fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failLock: true}
		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-lock-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should fail open and call handler
		if callCount != 1 {
			t.Errorf("Expected 1 handler call (fail open on lock error), got %d", callCount)
		}
		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}
	})

	t.Run("logs error but does not fail when store Unlock fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failUnlock: true}
		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-unlock-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should complete successfully even if unlock fails
		if callCount != 1 {
			t.Errorf("Expected 1 handler call, got %d", callCount)
		}
		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}
	})

	t.Run("logs error but does not fail when store Set fails", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		})

		store := &errorStore{failSet: true}
		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL:   time.Hour,
			Store: store,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-set-error")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		// Should complete successfully even if set fails
		if callCount != 1 {
			t.Errorf("Expected 1 handler call, got %d", callCount)
		}
		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}
	})
}

func TestIdempotency_PanicRecovery(t *testing.T) {
	t.Run("unlocks store when handler panics", func(t *testing.T) {
		store := NewIdempotencyMemoryStore(100)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("handler panic")
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !locked {
			t.Error("Lock should succeed after panic recovery (unlock was called)")
		}
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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
		if w2.Header().Get(httpx.HeaderConnection) != "" {
			t.Error("Connection header should not be replayed (hop-by-hop)")
		}
		if w2.Header().Get(httpx.HeaderKeepAlive) != "" {
			t.Error("Keep-Alive header should not be replayed (hop-by-hop)")
		}
		// Custom header should be replayed
		if w2.Header().Get("X-Custom") != "value" {
			t.Error("X-Custom header should be replayed")
		}
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
			TTL: time.Hour,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte(`{"amount":100}`)))
		req.Header.Set(httpx.HeaderIdempotencyKey, "key-idempotent")
		w := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201 (first WriteHeader), got %d", w.Code)
		}
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

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 1 {
			t.Errorf("Expected 1 handler call for allowed path (cached), got %d", callCount)
		}
		if w2.Header().Get(httpx.HeaderXIdempotencyReplay) != "true" {
			t.Error("Replayed request should have X-Idempotency-Replay header")
		}

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

		if callCount != 3 {
			t.Errorf("Expected 3 handler calls (non-allowed path not cached), got %d", callCount)
		}
	})

	t.Run("included paths with prefix match", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusOK)
		})

		idempotencyMiddleware := Idempotency(config.IdempotencyConfig{
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

		if callCount != 1 {
			t.Errorf("Expected 1 handler call for prefix match, got %d", callCount)
		}

		// Request outside prefix - should not be cached
		req3 := httptest.NewRequest(http.MethodPost, "/health", bytes.NewReader([]byte(`{}`)))
		req3.Header.Set(httpx.HeaderIdempotencyKey, "key-health")
		w3 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w3, req3)

		req4 := httptest.NewRequest(http.MethodPost, "/health", bytes.NewReader([]byte(`{}`)))
		req4.Header.Set(httpx.HeaderIdempotencyKey, "key-health")
		w4 := httptest.NewRecorder()
		idempotencyMiddleware(handler).ServeHTTP(w4, req4)

		if callCount != 3 {
			t.Errorf("Expected 3 handler calls (outside prefix not cached), got %d", callCount)
		}
	})
}

func TestIdempotency_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = Idempotency(config.IdempotencyConfig{
		TTL:           time.Hour,
		ExcludedPaths: []string{"/webhook"},
		IncludedPaths: []string{"/api"},
	})
}

package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
)

// BenchmarkIdempotency_CacheHit measures the overhead of a cache hit (store.Get + header replay).
// This is the common case in production and should be sub-microsecond.
func BenchmarkIdempotency_CacheHit(b *testing.B) {
	// Pre-populate the store
	store := NewIdempotencyMemoryStore(10000)
	ctx := context.Background()
	record := config.IdempotencyRecord{
		StatusCode: 201,
		Headers:    []string{"Content-Type", "application/json", "X-Custom", "value"},
		Body:       []byte(`{"id":"123"}`),
	}
	_ = store.Set(ctx, "test-key", record, time.Hour)

	mw := Idempotency(config.IdempotencyConfig{
		TTL:   time.Hour,
		Store: store,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"handler":"should not reach"}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewReader([]byte(`{}`)))
	req.Header.Set(httpx.HeaderIdempotencyKey, "test-key")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 201 {
			b.Fatalf("expected 201, got %d", w.Code)
		}
	}
}

// BenchmarkIdempotency_StoreGet isolates the store.Get call (RWMutex read lock + map lookup).
func BenchmarkIdempotency_StoreGet(b *testing.B) {
	store := NewIdempotencyMemoryStore(10000)
	ctx := context.Background()
	record := config.IdempotencyRecord{
		StatusCode: 201,
		Headers:    []string{"Content-Type", "application/json"},
		Body:       []byte(`{"id":"123"}`),
	}
	_ = store.Set(ctx, "test-key", record, time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = store.Get(ctx, "test-key")
	}
}

// BenchmarkIdempotency_HeaderReplay isolates header replay from cached record.
// Uses flat slice format [key1, val1, key2, val2, ...]
func BenchmarkIdempotency_HeaderReplay(b *testing.B) {
	record := config.IdempotencyRecord{
		StatusCode: 201,
		Headers: []string{
			"Content-Type", "application/json",
			"X-Custom", "value",
			"Cache-Control", "no-cache",
			"X-Request-ID", "abc123",
		},
		Body: []byte(`{"id":"123"}`),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		// Replay headers from flat slice
		for j := 0; j < len(record.Headers)-1; j += 2 {
			w.Header().Add(record.Headers[j], record.Headers[j+1])
		}
		w.Header().Set(httpx.HeaderXIdempotencyReplay, "true")
		w.WriteHeader(record.StatusCode)
		_, _ = w.Write(record.Body)
	}
}

// BenchmarkAddJitter measures math/rand cost. Should be ~3ns with 0 allocs.
func BenchmarkAddJitter(b *testing.B) {
	base := 10 * time.Millisecond

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = addJitter(base)
	}
}

// BenchmarkIdempotency_CacheKeyGeneration measures simple string join key generation.
func BenchmarkIdempotency_CacheKeyGeneration(b *testing.B) {
	body := []byte(`{"amount":100,"currency":"USD","description":"Test payment"}`)
	idempotencyKey := "idempotency-key-123"
	method := http.MethodPost
	path := "/api/payments"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = idempotencyKey + ":" + method + ":" + path + ":" + string(body)
	}
}

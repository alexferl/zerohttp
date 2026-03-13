package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkCache_CacheHit measures the full cache hit path including
// store.Get + header replay + body write. This is the value proposition
// of the middleware.
func BenchmarkCache_CacheHit(b *testing.B) {
	// Pre-populate the store
	store := NewCacheMemoryStore(10000)
	ctx := context.Background()
	record := config.CacheRecord{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"data":"cached response"}`),
		ETag:       `"abc123"`,
	}
	_ = store.Set(ctx, "GET|/api/data|", record, time.Hour)

	mw := Cache(config.CacheConfig{
		DefaultTTL: time.Hour,
		Store:      store,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":"fresh response"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			b.Fatalf("expected 200, got %d", w.Code)
		}
	}
}

// BenchmarkCache_GenerateCacheKey measures the SHA-256 key generation.
// This runs on every request, hit or miss. If it's slow, consider a
// cheaper key format without hashing.
func BenchmarkCache_GenerateCacheKey(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/api/users?page=1&limit=100", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US")

	vary := []string{"Accept", "Accept-Language"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = generateCacheKey(req, vary)
	}
}

// BenchmarkCache_StoreGet measures CacheMemoryStore.Get performance.
// Note: This uses sync.Mutex (not RWMutex) due to LRU tracking.
func BenchmarkCache_StoreGet(b *testing.B) {
	store := NewCacheMemoryStore(10000)
	ctx := context.Background()
	record := config.CacheRecord{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"data":"test"}`),
	}
	_ = store.Set(ctx, "test-key", record, time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = store.Get(ctx, "test-key")
	}
}

// BenchmarkCache_StoreGetConcurrent measures p99 latency under
// high read concurrency. sync.Mutex can get noisy here.
func BenchmarkCache_StoreGetConcurrent(b *testing.B) {
	store := NewCacheMemoryStore(10000)
	ctx := context.Background()
	record := config.CacheRecord{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"data":"test"}`),
	}

	// Pre-populate with multiple keys to reduce contention
	for i := 0; i < 100; i++ {
		_ = store.Set(ctx, keyForIndex(i), record, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _, _ = store.Get(ctx, keyForIndex(i%100))
			i++
		}
	})
}

// BenchmarkCache_StoreSet measures write performance.
func BenchmarkCache_StoreSet(b *testing.B) {
	store := NewCacheMemoryStore(10000)
	ctx := context.Background()
	record := config.CacheRecord{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"data":"test"}`),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		key := keyForIndex(i)
		_ = store.Set(ctx, key, record, time.Hour)
	}
}

// BenchmarkCache_ConcurrentReadWrite measures performance under
// mixed read/write load.
func BenchmarkCache_ConcurrentReadWrite(b *testing.B) {
	store := NewCacheMemoryStore(1000)
	ctx := context.Background()
	record := config.CacheRecord{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"data":"test"}`),
	}

	// Pre-populate
	for i := 0; i < 500; i++ {
		_ = store.Set(ctx, keyForIndex(i), record, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()

	var writeCounter int
	var mu sync.Mutex

	b.RunParallel(func(pb *testing.PB) {
		localCounter := 0
		for pb.Next() {
			// 90% reads, 10% writes
			if localCounter%10 == 0 {
				mu.Lock()
				writeCounter++
				key := keyForIndex(writeCounter)
				mu.Unlock()
				_ = store.Set(ctx, key, record, time.Hour)
			} else {
				_, _, _ = store.Get(ctx, keyForIndex(localCounter%500))
			}
			localCounter++
		}
	})
}

func keyForIndex(i int) string {
	// Generate unique keys
	return string(rune('a' + (i % 26)))
}

package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

// TestInMemoryStore_TokenBucket tests token bucket algorithm in MemoryStore
func TestInMemoryStore_TokenBucket(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Second, 2, 100)
	ctx := context.Background()
	now := time.Now()

	// First request allowed
	allowed, remaining, _ := store.CheckAndRecord(ctx, "key1", now)
	if !allowed {
		t.Error("first request should be allowed")
	}
	if remaining != 1 {
		t.Errorf("expected remaining=1, got %d", remaining)
	}

	// Second request allowed
	allowed, remaining, _ = store.CheckAndRecord(ctx, "key1", now)
	if !allowed {
		t.Error("second request should be allowed")
	}
	if remaining != 0 {
		t.Errorf("expected remaining=0, got %d", remaining)
	}

	// Third request denied
	allowed, remaining, _ = store.CheckAndRecord(ctx, "key1", now)
	if allowed {
		t.Error("third request should be denied")
	}
	if remaining != 0 {
		t.Errorf("expected remaining=0, got %d", remaining)
	}
}

// TestInMemoryStore_FixedWindow tests fixed window algorithm in MemoryStore
func TestInMemoryStore_FixedWindow(t *testing.T) {
	store := NewMemoryStore(FixedWindow, time.Second, 3, 100)
	ctx := context.Background()
	now := time.Now()

	// Three requests allowed
	for i := 0; i < 3; i++ {
		allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Fourth request denied
	allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
	if allowed {
		t.Error("fourth request should be denied")
	}
}

// TestInMemoryStore_SlidingWindow tests sliding window algorithm in MemoryStore
func TestInMemoryStore_SlidingWindow(t *testing.T) {
	store := NewMemoryStore(SlidingWindow, 100*time.Millisecond, 2, 100)
	ctx := context.Background()
	now := time.Now()

	// Two requests allowed
	for i := 0; i < 2; i++ {
		allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Third request denied
	allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
	if allowed {
		t.Error("third request should be denied")
	}

	// Wait for window to expire
	time.Sleep(110 * time.Millisecond)

	// After expiration, request allowed again
	allowed, _, _ = store.CheckAndRecord(ctx, "key1", time.Now())
	if !allowed {
		t.Error("request after window expiry should be allowed")
	}
}

// TestInMemoryStore_MaxKeysEviction tests that oldest entries are evicted at limit
func TestInMemoryStore_MaxKeysEviction(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Minute, 10, 5)
	ctx := context.Background()
	now := time.Now()

	// Create 5 entries (at limit)
	for i := 0; i < 5; i++ {
		key := string(rune('a' + i))
		allowed, _, _ := store.CheckAndRecord(ctx, key, now)
		if !allowed {
			t.Errorf("initial request for key %s should be allowed", key)
		}
	}

	// Wait a bit and access first key to update its lastAccess
	time.Sleep(10 * time.Millisecond)
	now = now.Add(10 * time.Millisecond)
	store.CheckAndRecord(ctx, "a", now)

	// Add 6th key - should evict the oldest (b, since a was just accessed)
	allowed, _, _ := store.CheckAndRecord(ctx, "f", now.Add(10*time.Millisecond))
	if !allowed {
		t.Error("request for new key should be allowed (eviction should happen)")
	}

	// Key "b" should have been evicted and treated as new
	allowed, _, _ = store.CheckAndRecord(ctx, "b", now.Add(20*time.Millisecond))
	if !allowed {
		t.Error("key 'b' should have been evicted and treated as new")
	}
}

// TestInMemoryStore_Expiration tests that expired entries are cleaned up
func TestInMemoryStore_Expiration(t *testing.T) {
	window := 100 * time.Millisecond
	store := NewMemoryStore(TokenBucket, window, 2, 100)
	ctx := context.Background()
	now := time.Now()

	// Use up tokens
	store.CheckAndRecord(ctx, "key1", now)
	store.CheckAndRecord(ctx, "key1", now)

	// Request denied
	allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
	if allowed {
		t.Error("request should be denied (no tokens)")
	}

	// Wait for entry to expire
	time.Sleep(110 * time.Millisecond)

	// After expiration, should be treated as new entry with fresh tokens
	allowed, remaining, _ := store.CheckAndRecord(ctx, "key1", time.Now())
	if !allowed {
		t.Error("request after expiration should be allowed (new entry)")
	}
	if remaining != 1 {
		t.Errorf("expected remaining=1 after expiration, got %d", remaining)
	}
}

// TestInMemoryStore_MultipleKeys tests isolation between keys
func TestInMemoryStore_MultipleKeys(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Second, 2, 100)
	ctx := context.Background()
	now := time.Now()

	// Exhaust tokens for key1
	store.CheckAndRecord(ctx, "key1", now)
	store.CheckAndRecord(ctx, "key1", now)

	// key2 should still have tokens
	allowed, remaining, _ := store.CheckAndRecord(ctx, "key2", now)
	if !allowed {
		t.Error("key2 first request should be allowed")
	}
	if remaining != 1 {
		t.Errorf("expected remaining=1 for key2, got %d", remaining)
	}

	// key1 should still be denied
	allowed, _, _ = store.CheckAndRecord(ctx, "key1", now)
	if allowed {
		t.Error("key1 should still be denied")
	}
}

// TestInMemoryStore_DefaultMaxKeys tests default max keys (0 = 10000)
func TestInMemoryStore_DefaultMaxKeys(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Second, 10, 0)

	if store.maxKeys != 10000 {
		t.Errorf("expected default maxKeys=10000, got %d", store.maxKeys)
	}
}

// mockStore is a test implementation of Store
type mockStore struct {
	checkFunc func(ctx context.Context, key string, now time.Time) (bool, int, time.Time)
}

func (m *mockStore) CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	return m.checkFunc(ctx, key, now)
}

func (m *mockStore) Close() error {
	return nil
}

// TestCustomStore tests using a custom store implementation
func TestCustomStore(t *testing.T) {
	resetTime := time.Now().Add(time.Minute)
	mock := &mockStore{
		checkFunc: func(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
			if key == "allowed" {
				return true, 5, resetTime
			}
			return false, 0, resetTime
		},
	}

	mw := New(Config{
		Store:      mock,
		Rate:       10,
		Window:     time.Minute,
		StatusCode: http.StatusTooManyRequests,
		Message:    "rate limited",
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test allowed key
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "allowed"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 for allowed key, got %d", w1.Code)
	}

	// Test denied key
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "denied"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for denied key, got %d", w2.Code)
	}
}

// TestInMemoryStore_ResetTime tests that reset time is calculated correctly
func TestInMemoryStore_ResetTime(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Minute, 60, 100)
	ctx := context.Background()
	now := time.Now()

	// Use up half the tokens
	for i := 0; i < 30; i++ {
		store.CheckAndRecord(ctx, "key1", now)
	}

	// Get reset time for remaining tokens
	_, remaining, resetTime := store.CheckAndRecord(ctx, "key1", now)
	if remaining != 29 {
		t.Errorf("expected remaining=29, got %d", remaining)
	}

	// Reset time should be in the future
	if !resetTime.After(now) {
		t.Error("resetTime should be after now")
	}
}

// TestRateLimit_WithMaxKeysConfig tests MaxKeys config option
func TestRateLimit_WithMaxKeysConfig(t *testing.T) {
	mw := New(Config{
		Rate:      10,
		Window:    time.Minute,
		Algorithm: TokenBucket,
		MaxKeys:   3,
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create 3 different keys
	for i := 0; i < 3; i++ {
		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		req.RemoteAddr = "192.168.1." + string(rune('1'+i)) + ":12345"
		w := zhtest.Serve(handler, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Create 4th key - should still work (eviction happens)
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "192.168.1.100:12345"
	w := zhtest.Serve(handler, req)
	if w.Code != http.StatusOK {
		t.Error("4th key should be allowed (eviction)")
	}
}

// TestInMemoryStore_MaxKeysEviction_FixedWindow tests eviction for fixed window
func TestInMemoryStore_MaxKeysEviction_FixedWindow(t *testing.T) {
	store := NewMemoryStore(FixedWindow, time.Minute, 100, 3)
	ctx := context.Background()
	now := time.Now()

	// Create 3 entries (each uses 1 token)
	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		store.CheckAndRecord(ctx, key, now)
	}

	// Wait and access first key to update lastAccess (uses another token)
	time.Sleep(10 * time.Millisecond)
	now = now.Add(10 * time.Millisecond)
	store.CheckAndRecord(ctx, "a", now)

	// Add 4th key - should evict oldest (b)
	store.CheckAndRecord(ctx, "d", now.Add(10*time.Millisecond))

	// Key "b" should have been evicted
	// Check by seeing if it's treated as new (allowed with full count)
	allowed, remaining, _ := store.CheckAndRecord(ctx, "b", now.Add(20*time.Millisecond))
	if !allowed {
		t.Error("key 'b' should have been evicted and treated as new")
	}
	// After eviction, key 'b' is recreated with count=1, so remaining = 100 - 1 = 99
	// Even if key 'a' was accessed twice (count=2), its lastAccess is updated so 'b' is evicted
	if remaining != 99 {
		t.Errorf("expected remaining=99 for evicted key, got %d", remaining)
	}
}

// TestInMemoryStore_MaxKeysEviction_SlidingWindow tests eviction for sliding window
func TestInMemoryStore_MaxKeysEviction_SlidingWindow(t *testing.T) {
	store := NewMemoryStore(SlidingWindow, time.Minute, 100, 3)
	ctx := context.Background()
	now := time.Now()

	// Create 3 entries
	for i := 0; i < 3; i++ {
		key := string(rune('a' + i))
		store.CheckAndRecord(ctx, key, now)
	}

	// Wait and access first key to update lastAccess
	time.Sleep(10 * time.Millisecond)
	now = now.Add(10 * time.Millisecond)
	store.CheckAndRecord(ctx, "a", now)

	// Add 4th key - should evict oldest (b)
	store.CheckAndRecord(ctx, "d", now.Add(10*time.Millisecond))

	// Key "b" should have been evicted
	// Just check that it's allowed (not rate limited)
	allowed, _, _ := store.CheckAndRecord(ctx, "b", now.Add(20*time.Millisecond))
	if !allowed {
		t.Error("key 'b' should have been evicted and treated as new")
	}
}

func TestInMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Second, 10, 100)

	if err := store.Close(); err != nil {
		t.Errorf("Unexpected error closing store: %v", err)
	}
}

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
	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 1, remaining)

	// Second request allowed
	allowed, remaining, _ = store.CheckAndRecord(ctx, "key1", now)
	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 0, remaining)

	// Third request denied
	allowed, remaining, _ = store.CheckAndRecord(ctx, "key1", now)
	zhtest.AssertFalse(t, allowed)
	zhtest.AssertEqual(t, 0, remaining)
}

// TestInMemoryStore_FixedWindow tests fixed window algorithm in MemoryStore
func TestInMemoryStore_FixedWindow(t *testing.T) {
	store := NewMemoryStore(FixedWindow, time.Second, 3, 100)
	ctx := context.Background()
	now := time.Now()

	// Three requests allowed
	for i := 0; i < 3; i++ {
		allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
		zhtest.AssertTrue(t, allowed)
	}

	// Fourth request denied
	allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
	zhtest.AssertFalse(t, allowed)
}

// TestInMemoryStore_SlidingWindow tests sliding window algorithm in MemoryStore
func TestInMemoryStore_SlidingWindow(t *testing.T) {
	store := NewMemoryStore(SlidingWindow, 100*time.Millisecond, 2, 100)
	ctx := context.Background()
	now := time.Now()

	// Two requests allowed
	for i := 0; i < 2; i++ {
		allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
		zhtest.AssertTrue(t, allowed)
	}

	// Third request denied
	allowed, _, _ := store.CheckAndRecord(ctx, "key1", now)
	zhtest.AssertFalse(t, allowed)

	// Wait for window to expire
	time.Sleep(110 * time.Millisecond)

	// After expiration, request allowed again
	allowed, _, _ = store.CheckAndRecord(ctx, "key1", time.Now())
	zhtest.AssertTrue(t, allowed)
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
		zhtest.AssertTrue(t, allowed)
	}

	// Wait a bit and access first key to update its lastAccess
	time.Sleep(10 * time.Millisecond)
	now = now.Add(10 * time.Millisecond)
	store.CheckAndRecord(ctx, "a", now)

	// Add 6th key - should evict the oldest (b, since a was just accessed)
	allowed, _, _ := store.CheckAndRecord(ctx, "f", now.Add(10*time.Millisecond))
	zhtest.AssertTrue(t, allowed)

	// Key "b" should have been evicted and treated as new
	allowed, _, _ = store.CheckAndRecord(ctx, "b", now.Add(20*time.Millisecond))
	zhtest.AssertTrue(t, allowed)
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
	zhtest.AssertFalse(t, allowed)

	// Wait for entry to expire
	time.Sleep(110 * time.Millisecond)

	// After expiration, should be treated as new entry with fresh tokens
	allowed, remaining, _ := store.CheckAndRecord(ctx, "key1", time.Now())
	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 1, remaining)
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
	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 1, remaining)

	// key1 should still be denied
	allowed, _, _ = store.CheckAndRecord(ctx, "key1", now)
	zhtest.AssertFalse(t, allowed)
}

// TestInMemoryStore_DefaultMaxKeys tests default max keys (0 = 10000)
func TestInMemoryStore_DefaultMaxKeys(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Second, 10, 0)
	zhtest.AssertEqual(t, 10000, store.maxKeys)
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
	zhtest.AssertEqual(t, http.StatusOK, w1.Code)

	// Test denied key
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "denied"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	zhtest.AssertEqual(t, http.StatusTooManyRequests, w2.Code)
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
	zhtest.AssertEqual(t, 29, remaining)

	// Reset time should be in the future
	zhtest.AssertTrue(t, resetTime.After(now))
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
		zhtest.AssertEqual(t, http.StatusOK, w.Code)
	}

	// Create 4th key - should still work (eviction happens)
	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	req.RemoteAddr = "192.168.1.100:12345"
	w := zhtest.Serve(handler, req)
	zhtest.AssertEqual(t, http.StatusOK, w.Code)
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
	zhtest.AssertTrue(t, allowed)
	// After eviction, key 'b' is recreated with count=1, so remaining = 100 - 1 = 99
	// Even if key 'a' was accessed twice (count=2), its lastAccess is updated so 'b' is evicted
	zhtest.AssertEqual(t, 99, remaining)
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
	zhtest.AssertTrue(t, allowed)
}

func TestInMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore(TokenBucket, time.Second, 10, 100)

	err := store.Close()
	zhtest.AssertNoError(t, err)
}

package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

func TestCacheMemoryStore(t *testing.T) {
	ctx := context.Background()

	t.Run("basic get and set", func(t *testing.T) {
		store := NewCacheMemoryStore(100)

		record := config.CacheRecord{
			StatusCode: 200,
			Body:       []byte("test"),
		}

		if err := store.Set(ctx, "key1", record, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		retrieved, found, err := store.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !found {
			t.Error("Expected to find key1")
		}
		if string(retrieved.Body) != "test" {
			t.Errorf("Expected body 'test', got %q", string(retrieved.Body))
		}
	})

	t.Run("expired entry not returned", func(t *testing.T) {
		store := NewCacheMemoryStore(100)

		record := config.CacheRecord{
			StatusCode: 200,
			Body:       []byte("test"),
		}

		if err := store.Set(ctx, "key1", record, 1*time.Millisecond); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}
		time.Sleep(2 * time.Millisecond)

		_, found, err := store.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if found {
			t.Error("Expected expired entry to not be found")
		}
	})

	t.Run("LRU eviction", func(t *testing.T) {
		store := NewCacheMemoryStore(2)

		if err := store.Set(ctx, "key1", config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}
		if err := store.Set(ctx, "key2", config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}
		if err := store.Set(ctx, "key3", config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		// key1 should be evicted
		_, found, _ := store.Get(ctx, "key1")
		if found {
			t.Error("Expected key1 to be evicted")
		}

		// key2 and key3 should exist
		_, found, _ = store.Get(ctx, "key2")
		if !found {
			t.Error("Expected key2 to exist")
		}
		_, found, _ = store.Get(ctx, "key3")
		if !found {
			t.Error("Expected key3 to exist")
		}
	})

	t.Run("update existing key moves to front", func(t *testing.T) {
		store := NewCacheMemoryStore(2)

		if err := store.Set(ctx, "key1", config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}
		if err := store.Set(ctx, "key2", config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		// Access key1 to make it most recently used
		_, _, _ = store.Get(ctx, "key1")

		// Add key3 - key2 should be evicted (least recently used)
		if err := store.Set(ctx, "key3", config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		// key1 should still exist (was accessed)
		_, found, _ := store.Get(ctx, "key1")
		if !found {
			t.Error("Expected key1 to exist (was accessed recently)")
		}

		// key2 should be evicted
		_, found, _ = store.Get(ctx, "key2")
		if found {
			t.Error("Expected key2 to be evicted (was not accessed)")
		}
	})

	t.Run("unlimited capacity when maxEntries is 0", func(t *testing.T) {
		store := NewCacheMemoryStore(0)

		// Add many entries
		for i := 0; i < 100; i++ {
			if err := store.Set(ctx, string(rune('a'+i)), config.CacheRecord{StatusCode: 200}, time.Minute); err != nil {
				t.Errorf("Unexpected error setting key: %v", err)
			}
		}

		// All should exist
		for i := 0; i < 100; i++ {
			_, found, _ := store.Get(ctx, string(rune('a'+i)))
			if !found {
				t.Errorf("Expected key %c to exist", 'a'+i)
			}
		}
	})

	t.Run("update preserves expiry", func(t *testing.T) {
		store := NewCacheMemoryStore(100)

		record1 := config.CacheRecord{
			StatusCode: 200,
			Body:       []byte("v1"),
		}
		if err := store.Set(ctx, "key1", record1, 1*time.Millisecond); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		// Wait for original to nearly expire
		time.Sleep(500 * time.Microsecond)

		// Update with new TTL
		record2 := config.CacheRecord{
			StatusCode: 200,
			Body:       []byte("v2"),
		}
		if err := store.Set(ctx, "key1", record2, time.Minute); err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		// Wait for original TTL to expire
		time.Sleep(1 * time.Millisecond)

		// Should still exist due to updated TTL
		retrieved, found, _ := store.Get(ctx, "key1")
		if !found {
			t.Error("Expected key1 to exist after TTL update")
		}
		if string(retrieved.Body) != "v2" {
			t.Errorf("Expected body 'v2', got %q", string(retrieved.Body))
		}
	})
}

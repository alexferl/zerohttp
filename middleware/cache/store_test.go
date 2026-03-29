package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestCacheMemoryStore(t *testing.T) {
	ctx := context.Background()

	t.Run("basic get and set", func(t *testing.T) {
		store := NewMemoryStore(100)

		record := Record{
			StatusCode: 200,
			Body:       []byte("test"),
		}

		zhtest.AssertNoError(t, store.Set(ctx, "key1", record, time.Minute))

		retrieved, found, err := store.Get(ctx, "key1")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, "test", string(retrieved.Body))
	})

	t.Run("expired entry not returned", func(t *testing.T) {
		store := NewMemoryStore(100)

		record := Record{
			StatusCode: 200,
			Body:       []byte("test"),
		}

		zhtest.AssertNoError(t, store.Set(ctx, "key1", record, 1*time.Millisecond))
		time.Sleep(2 * time.Millisecond)

		_, found, err := store.Get(ctx, "key1")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})

	t.Run("LRU eviction", func(t *testing.T) {
		store := NewMemoryStore(2)

		zhtest.AssertNoError(t, store.Set(ctx, "key1", Record{StatusCode: 200}, time.Minute))
		zhtest.AssertNoError(t, store.Set(ctx, "key2", Record{StatusCode: 200}, time.Minute))
		zhtest.AssertNoError(t, store.Set(ctx, "key3", Record{StatusCode: 200}, time.Minute))

		// key1 should be evicted
		_, found, _ := store.Get(ctx, "key1")
		zhtest.AssertFalse(t, found)

		// key2 and key3 should exist
		_, found, _ = store.Get(ctx, "key2")
		zhtest.AssertTrue(t, found)
		_, found, _ = store.Get(ctx, "key3")
		zhtest.AssertTrue(t, found)
	})

	t.Run("update existing key moves to front", func(t *testing.T) {
		store := NewMemoryStore(2)

		zhtest.AssertNoError(t, store.Set(ctx, "key1", Record{StatusCode: 200}, time.Minute))
		zhtest.AssertNoError(t, store.Set(ctx, "key2", Record{StatusCode: 200}, time.Minute))

		// Access key1 to make it most recently used
		_, _, _ = store.Get(ctx, "key1")

		// Add key3 - key2 should be evicted (least recently used)
		zhtest.AssertNoError(t, store.Set(ctx, "key3", Record{StatusCode: 200}, time.Minute))

		// key1 should still exist (was accessed)
		_, found, _ := store.Get(ctx, "key1")
		zhtest.AssertTrue(t, found)

		// key2 should be evicted
		_, found, _ = store.Get(ctx, "key2")
		zhtest.AssertFalse(t, found)
	})

	t.Run("unlimited capacity when maxEntries is 0", func(t *testing.T) {
		store := NewMemoryStore(0)

		// Add many entries
		for i := 0; i < 100; i++ {
			zhtest.AssertNoError(t, store.Set(ctx, string(rune('a'+i)), Record{StatusCode: 200}, time.Minute))
		}

		// All should exist
		for i := 0; i < 100; i++ {
			_, found, _ := store.Get(ctx, string(rune('a'+i)))
			zhtest.AssertTrue(t, found)
		}
	})

	t.Run("update preserves expiry", func(t *testing.T) {
		store := NewMemoryStore(100)

		record1 := Record{
			StatusCode: 200,
			Body:       []byte("v1"),
		}
		zhtest.AssertNoError(t, store.Set(ctx, "key1", record1, 1*time.Millisecond))

		// Wait for original to nearly expire
		time.Sleep(500 * time.Microsecond)

		// Update with new TTL
		record2 := Record{
			StatusCode: 200,
			Body:       []byte("v2"),
		}
		zhtest.AssertNoError(t, store.Set(ctx, "key1", record2, time.Minute))

		// Wait for original TTL to expire
		time.Sleep(1 * time.Millisecond)

		// Should still exist due to updated TTL
		retrieved, found, _ := store.Get(ctx, "key1")
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, "v2", string(retrieved.Body))
	})
}

func TestCacheMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(100)

	// Set a value first
	record := Record{
		StatusCode: 200,
		Body:       []byte("test"),
	}
	zhtest.AssertNoError(t, store.Set(ctx, "key1", record, time.Minute))

	// Verify it exists
	_, found, _ := store.Get(ctx, "key1")
	zhtest.AssertTrue(t, found)

	// Delete it
	zhtest.AssertNoError(t, store.Delete(ctx, "key1"))

	// Verify it's gone
	_, found, _ = store.Get(ctx, "key1")
	zhtest.AssertFalse(t, found)

	// Deleting non-existent key should not error
	zhtest.AssertNoError(t, store.Delete(ctx, "nonexistent"))
}

func TestCacheMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore(100)

	zhtest.AssertNoError(t, store.Close())
}

package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestIdempotencyMemoryStore(t *testing.T) {
	t.Run("basic get and set", func(t *testing.T) {
		store := NewMemoryStore(100)

		record := Record{
			StatusCode: 201,
			Headers:    []string{"Content-Type", "application/json"},
			Body:       []byte(`{"id":"123"}`),
			CreatedAt:  time.Now().UTC(),
		}

		err := store.Set(context.Background(), "key1", record, time.Hour)
		zhtest.AssertNoError(t, err)

		retrieved, found, err := store.Get(context.Background(), "key1")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, `{"id":"123"}`, string(retrieved.Body))
	})

	t.Run("expired entry not returned", func(t *testing.T) {
		store := NewMemoryStore(100)

		record := Record{
			StatusCode: 201,
			Body:       []byte(`test`),
		}

		err := store.Set(context.Background(), "key1", record, 1*time.Millisecond)
		zhtest.AssertNoError(t, err)

		time.Sleep(2 * time.Millisecond)

		_, found, err := store.Get(context.Background(), "key1")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})

	t.Run("not found returns false", func(t *testing.T) {
		store := NewMemoryStore(100)

		_, found, err := store.Get(context.Background(), "nonexistent-key")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
	})
}

func TestIdempotencyMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore(100)

	err := store.Close()
	zhtest.AssertNoError(t, err)
}

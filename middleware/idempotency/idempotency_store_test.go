package idempotency

import (
	"context"
	"testing"
	"time"
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
		if err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		retrieved, found, err := store.Get(context.Background(), "key1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !found {
			t.Error("Expected to find key1")
		}
		if string(retrieved.Body) != `{"id":"123"}` {
			t.Errorf("Expected body '{\"id\":\"123\"}', got %q", string(retrieved.Body))
		}
	})

	t.Run("expired entry not returned", func(t *testing.T) {
		store := NewMemoryStore(100)

		record := Record{
			StatusCode: 201,
			Body:       []byte(`test`),
		}

		err := store.Set(context.Background(), "key1", record, 1*time.Millisecond)
		if err != nil {
			t.Errorf("Unexpected error setting key: %v", err)
		}

		time.Sleep(2 * time.Millisecond)

		_, found, err := store.Get(context.Background(), "key1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if found {
			t.Error("Expected expired entry to not be found")
		}
	})

	t.Run("not found returns false", func(t *testing.T) {
		store := NewMemoryStore(100)

		_, found, err := store.Get(context.Background(), "nonexistent-key")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if found {
			t.Error("Expected not found for nonexistent key")
		}
	})
}

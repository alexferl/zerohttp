package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultIdempotencyConfig(t *testing.T) {
	t.Run("default values are set", func(t *testing.T) {
		cfg := DefaultConfig

		zhtest.AssertEqual(t, "Idempotency-Key", cfg.HeaderName)
		zhtest.AssertEqual(t, 24*time.Hour, cfg.TTL)
		zhtest.AssertEqual(t, 1024*1024, cfg.MaxBodySize)
		zhtest.AssertFalse(t, cfg.Required)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
		zhtest.AssertNil(t, cfg.Store)
		zhtest.AssertEqual(t, 10000, cfg.MaxKeys)
		zhtest.AssertEqual(t, 10*time.Millisecond, cfg.LockRetryInterval)
		zhtest.AssertEqual(t, 300, cfg.LockMaxRetries)
		zhtest.AssertEqual(t, 500*time.Millisecond, cfg.LockMaxInterval)
		zhtest.AssertEqual(t, 2.0, cfg.LockBackoffMultiplier)
	})
}

func TestIdempotencyRecord(t *testing.T) {
	t.Run("idempotency record fields", func(t *testing.T) {
		record := Record{
			StatusCode: 201,
			Headers:    []string{"Content-Type", "application/json"},
			Body:       []byte(`{"id":"123"}`),
			CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		zhtest.AssertEqual(t, 201, record.StatusCode)
		zhtest.AssertEqual(t, `{"id":"123"}`, string(record.Body))
	})
}

func TestIdempotencyConfigCustomization(t *testing.T) {
	t.Run("custom config values", func(t *testing.T) {
		customStore := &mockIdempotencyStore{}

		cfg := Config{
			HeaderName:    "X-Idempotency-Key",
			TTL:           time.Hour,
			MaxBodySize:   512 * 1024,
			Store:         customStore,
			Required:      true,
			ExcludedPaths: []string{"/webhook", "/callback"},
			IncludedPaths: []string{"/api/public"},
			MaxKeys:       5000,
		}

		zhtest.AssertEqual(t, "X-Idempotency-Key", cfg.HeaderName)
		zhtest.AssertEqual(t, time.Hour, cfg.TTL)
		zhtest.AssertEqual(t, 512*1024, cfg.MaxBodySize)
		zhtest.AssertEqual(t, customStore, cfg.Store)
		zhtest.AssertTrue(t, cfg.Required)
		zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, 5000, cfg.MaxKeys)
	})
}

func TestIdempotencyConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			HeaderName:    DefaultConfig.HeaderName,
			TTL:           DefaultConfig.TTL,
			MaxBodySize:   DefaultConfig.MaxBodySize,
			MaxKeys:       DefaultConfig.MaxKeys,
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})
}

// mockIdempotencyStore is a minimal implementation for testing
type mockIdempotencyStore struct{}

func (m *mockIdempotencyStore) Get(ctx context.Context, key string) (Record, bool, error) {
	return Record{}, false, nil
}

func (m *mockIdempotencyStore) Set(ctx context.Context, key string, record Record, ttl time.Duration) error {
	return nil
}

func (m *mockIdempotencyStore) Lock(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func (m *mockIdempotencyStore) Unlock(ctx context.Context, key string) error {
	return nil
}

func (m *mockIdempotencyStore) Close() error {
	return nil
}

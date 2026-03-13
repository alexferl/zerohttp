package config

import (
	"context"
	"testing"
	"time"
)

func TestDefaultIdempotencyConfig(t *testing.T) {
	t.Run("default values are set", func(t *testing.T) {
		cfg := DefaultIdempotencyConfig

		if cfg.HeaderName != "Idempotency-Key" {
			t.Errorf("expected HeaderName 'Idempotency-Key', got %q", cfg.HeaderName)
		}

		if cfg.TTL != 24*time.Hour {
			t.Errorf("expected TTL 24h, got %v", cfg.TTL)
		}

		if cfg.MaxBodySize != 1024*1024 {
			t.Errorf("expected MaxBodySize 1MB, got %d", cfg.MaxBodySize)
		}

		if cfg.Required {
			t.Error("expected Required to be false by default")
		}

		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected ExemptPaths to be empty, got %v", cfg.ExemptPaths)
		}

		if cfg.Store != nil {
			t.Error("expected Store to be nil by default")
		}

		if cfg.MaxKeys != 10000 {
			t.Errorf("expected MaxKeys 10000, got %d", cfg.MaxKeys)
		}

		if cfg.LockRetryInterval != 10*time.Millisecond {
			t.Errorf("expected LockRetryInterval 10ms, got %v", cfg.LockRetryInterval)
		}

		if cfg.LockMaxRetries != 300 {
			t.Errorf("expected LockMaxRetries 300, got %d", cfg.LockMaxRetries)
		}

		if cfg.LockMaxInterval != 500*time.Millisecond {
			t.Errorf("expected LockMaxInterval 500ms, got %v", cfg.LockMaxInterval)
		}

		if cfg.LockBackoffMultiplier != 2.0 {
			t.Errorf("expected LockBackoffMultiplier 2.0, got %f", cfg.LockBackoffMultiplier)
		}
	})
}

func TestIdempotencyRecord(t *testing.T) {
	t.Run("idempotency record fields", func(t *testing.T) {
		record := IdempotencyRecord{
			StatusCode: 201,
			Headers:    []string{"Content-Type", "application/json"},
			Body:       []byte(`{"id":"123"}`),
			CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		if record.StatusCode != 201 {
			t.Errorf("expected StatusCode 201, got %d", record.StatusCode)
		}

		if string(record.Body) != `{"id":"123"}` {
			t.Errorf("expected Body %q, got %q", `{"id":"123"}`, string(record.Body))
		}
	})
}

func TestIdempotencyConfigCustomization(t *testing.T) {
	t.Run("custom config values", func(t *testing.T) {
		customStore := &mockIdempotencyStore{}

		cfg := IdempotencyConfig{
			HeaderName:  "X-Idempotency-Key",
			TTL:         time.Hour,
			MaxBodySize: 512 * 1024,
			Store:       customStore,
			Required:    true,
			ExemptPaths: []string{"/webhook", "/callback"},
			MaxKeys:     5000,
		}

		if cfg.HeaderName != "X-Idempotency-Key" {
			t.Errorf("expected HeaderName 'X-Idempotency-Key', got %q", cfg.HeaderName)
		}

		if cfg.TTL != time.Hour {
			t.Errorf("expected TTL 1h, got %v", cfg.TTL)
		}

		if cfg.MaxBodySize != 512*1024 {
			t.Errorf("expected MaxBodySize 512KB, got %d", cfg.MaxBodySize)
		}

		if cfg.Store != customStore {
			t.Error("expected custom store to be set")
		}

		if !cfg.Required {
			t.Error("expected Required to be true")
		}

		if len(cfg.ExemptPaths) != 2 {
			t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
		}

		if cfg.MaxKeys != 5000 {
			t.Errorf("expected MaxKeys 5000, got %d", cfg.MaxKeys)
		}
	})
}

// mockIdempotencyStore is a minimal implementation for testing
type mockIdempotencyStore struct{}

func (m *mockIdempotencyStore) Get(ctx context.Context, key string) (IdempotencyRecord, bool, error) {
	return IdempotencyRecord{}, false, nil
}

func (m *mockIdempotencyStore) Set(ctx context.Context, key string, record IdempotencyRecord, ttl time.Duration) error {
	return nil
}

func (m *mockIdempotencyStore) Lock(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func (m *mockIdempotencyStore) Unlock(ctx context.Context, key string) error {
	return nil
}

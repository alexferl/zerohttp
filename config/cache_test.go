package config

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCacheConfig(t *testing.T) {
	t.Run("default values are set", func(t *testing.T) {
		cfg := DefaultCacheConfig

		if cfg.CacheControl != "private, max-age=60" {
			t.Errorf("expected CacheControl 'private, max-age=60', got %q", cfg.CacheControl)
		}

		if cfg.DefaultTTL != time.Minute {
			t.Errorf("expected DefaultTTL 1m, got %v", cfg.DefaultTTL)
		}

		if cfg.MaxBodySize != 10*1024*1024 {
			t.Errorf("expected MaxBodySize 10MB, got %d", cfg.MaxBodySize)
		}

		if cfg.MaxEntries != 10000 {
			t.Errorf("expected MaxEntries 10000, got %d", cfg.MaxEntries)
		}

		if !cfg.ETag {
			t.Error("expected ETag to be true by default")
		}

		if !cfg.LastModified {
			t.Error("expected LastModified to be true by default")
		}

		expectedVary := []string{"Accept", "Accept-Encoding", "Accept-Language"}
		if len(cfg.Vary) != len(expectedVary) {
			t.Errorf("expected Vary %v, got %v", expectedVary, cfg.Vary)
		}
		for i, v := range expectedVary {
			if cfg.Vary[i] != v {
				t.Errorf("expected Vary[%d] %q, got %q", i, v, cfg.Vary[i])
			}
		}

		if cfg.Store != nil {
			t.Error("expected Store to be nil by default")
		}

		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected ExemptPaths to be empty, got %v", cfg.ExemptPaths)
		}

		expectedStatusCodes := []int{200, 201, 204, 301, 302, 304, 307, 308}
		if len(cfg.StatusCodes) != len(expectedStatusCodes) {
			t.Errorf("expected %d status codes, got %d", len(expectedStatusCodes), len(cfg.StatusCodes))
		}
	})
}

func TestCacheRecord(t *testing.T) {
	t.Run("cache record fields", func(t *testing.T) {
		record := CacheRecord{
			StatusCode:   200,
			Headers:      map[string][]string{"Content-Type": {"application/json"}},
			Body:         []byte(`{"message":"hello"}`),
			ETag:         `"abc123"`,
			LastModified: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			VaryHeaders:  map[string]string{"Accept": "application/json"},
		}

		if record.StatusCode != 200 {
			t.Errorf("expected StatusCode 200, got %d", record.StatusCode)
		}

		if string(record.Body) != `{"message":"hello"}` {
			t.Errorf("expected Body %q, got %q", `{"message":"hello"}`, string(record.Body))
		}

		if record.ETag != `"abc123"` {
			t.Errorf("expected ETag %q, got %q", `"abc123"`, record.ETag)
		}
	})
}

func TestCacheConfigCustomization(t *testing.T) {
	t.Run("custom config values", func(t *testing.T) {
		customStore := &mockCacheStore{}

		cfg := CacheConfig{
			CacheControl: "public, max-age=3600",
			DefaultTTL:   time.Hour,
			MaxBodySize:  5 * 1024 * 1024,
			MaxEntries:   5000,
			ETag:         false,
			LastModified: false,
			Vary:         []string{"Accept"},
			Store:        customStore,
			ExemptPaths:  []string{"/api/live", "/health"},
			StatusCodes:  []int{200, 201},
		}

		if cfg.CacheControl != "public, max-age=3600" {
			t.Errorf("expected CacheControl 'public, max-age=3600', got %q", cfg.CacheControl)
		}

		if cfg.DefaultTTL != time.Hour {
			t.Errorf("expected DefaultTTL 1h, got %v", cfg.DefaultTTL)
		}

		if cfg.MaxBodySize != 5*1024*1024 {
			t.Errorf("expected MaxBodySize 5MB, got %d", cfg.MaxBodySize)
		}

		if cfg.MaxEntries != 5000 {
			t.Errorf("expected MaxEntries 5000, got %d", cfg.MaxEntries)
		}

		if cfg.ETag {
			t.Error("expected ETag to be false")
		}

		if cfg.LastModified {
			t.Error("expected LastModified to be false")
		}

		if cfg.Store != customStore {
			t.Error("expected custom store to be set")
		}

		if len(cfg.ExemptPaths) != 2 {
			t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
		}
	})
}

// mockCacheStore is a minimal implementation for testing
type mockCacheStore struct{}

func (m *mockCacheStore) Get(ctx context.Context, key string) (CacheRecord, bool, error) {
	return CacheRecord{}, false, nil
}

func (m *mockCacheStore) Set(ctx context.Context, key string, record CacheRecord, ttl time.Duration) error {
	return nil
}

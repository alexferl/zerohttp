package cache

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCacheConfig(t *testing.T) {
	t.Run("default values are set", func(t *testing.T) {
		cfg := DefaultConfig

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

		if len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected ExcludedPaths to be empty, got %v", cfg.ExcludedPaths)
		}

		if len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected IncludedPaths to be empty, got %v", cfg.IncludedPaths)
		}

		expectedStatusCodes := []int{200, 201, 204, 301, 302, 304, 307, 308}
		if len(cfg.StatusCodes) != len(expectedStatusCodes) {
			t.Errorf("expected %d status codes, got %d", len(expectedStatusCodes), len(cfg.StatusCodes))
		}
	})
}

func TestCacheRecord(t *testing.T) {
	t.Run("cache record fields", func(t *testing.T) {
		record := Record{
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

		cfg := Config{
			CacheControl:  "public, max-age=3600",
			DefaultTTL:    time.Hour,
			MaxBodySize:   5 * 1024 * 1024,
			MaxEntries:    5000,
			ETag:          false,
			LastModified:  false,
			Vary:          []string{"Accept"},
			Store:         customStore,
			ExcludedPaths: []string{"/api/live", "/health"},
			StatusCodes:   []int{200, 201},
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

		if len(cfg.ExcludedPaths) != 2 {
			t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
	})
}

func TestCacheConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			CacheControl:  DefaultConfig.CacheControl,
			DefaultTTL:    DefaultConfig.DefaultTTL,
			MaxBodySize:   DefaultConfig.MaxBodySize,
			MaxEntries:    DefaultConfig.MaxEntries,
			ETag:          DefaultConfig.ETag,
			LastModified:  DefaultConfig.LastModified,
			Vary:          DefaultConfig.Vary,
			StatusCodes:   DefaultConfig.StatusCodes,
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if cfg.IncludedPaths[0] != "/api/public" {
			t.Errorf("expected first allowed path to be /api/public, got %s", cfg.IncludedPaths[0])
		}
		if cfg.IncludedPaths[1] != "/health" {
			t.Errorf("expected second allowed path to be /health, got %s", cfg.IncludedPaths[1])
		}
	})

	t.Run("both excluded and included paths", func(t *testing.T) {
		excludedPaths := []string{"/api/live", "/health"}
		includedPaths := []string{"/api/public"}
		cfg := Config{
			CacheControl:  DefaultConfig.CacheControl,
			DefaultTTL:    DefaultConfig.DefaultTTL,
			MaxBodySize:   DefaultConfig.MaxBodySize,
			MaxEntries:    DefaultConfig.MaxEntries,
			ETag:          DefaultConfig.ETag,
			LastModified:  DefaultConfig.LastModified,
			Vary:          DefaultConfig.Vary,
			StatusCodes:   DefaultConfig.StatusCodes,
			ExcludedPaths: excludedPaths,
			IncludedPaths: includedPaths,
		}
		if len(cfg.ExcludedPaths) != 2 {
			t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if len(cfg.IncludedPaths) != 1 {
			t.Errorf("expected 1 allowed path, got %d", len(cfg.IncludedPaths))
		}
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: []string{},
		}
		if cfg.IncludedPaths == nil {
			t.Error("expected included paths slice to be initialized, not nil")
		}
		if len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected empty included paths slice, got %d entries", len(cfg.IncludedPaths))
		}
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: nil,
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
		}
	})
}

// mockCacheStore is a minimal implementation for testing
type mockCacheStore struct{}

func (m *mockCacheStore) Get(ctx context.Context, key string) (Record, bool, error) {
	return Record{}, false, nil
}

func (m *mockCacheStore) Set(ctx context.Context, key string, record Record, ttl time.Duration) error {
	return nil
}

func (m *mockCacheStore) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCacheStore) Close() error {
	return nil
}

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultCacheConfig(t *testing.T) {
	t.Run("default values are set", func(t *testing.T) {
		cfg := DefaultConfig

		zhtest.AssertEqual(t, "private, max-age=60", cfg.CacheControl)
		zhtest.AssertEqual(t, time.Minute, cfg.DefaultTTL)
		zhtest.AssertEqual(t, 10*1024*1024, cfg.MaxBodySize)
		zhtest.AssertEqual(t, 10000, cfg.MaxEntries)
		zhtest.AssertTrue(t, cfg.ETag)
		zhtest.AssertTrue(t, cfg.LastModified)

		expectedVary := []string{"Accept", "Accept-Encoding", "Accept-Language"}
		zhtest.AssertEqual(t, len(expectedVary), len(cfg.Vary))
		for i, v := range expectedVary {
			zhtest.AssertEqual(t, v, cfg.Vary[i])
		}

		zhtest.AssertNil(t, cfg.Store)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

		expectedStatusCodes := []int{200, 201, 204, 301, 302, 304, 307, 308}
		zhtest.AssertEqual(t, len(expectedStatusCodes), len(cfg.StatusCodes))
	})
}

func TestCacheRecord(t *testing.T) {
	t.Run("cache record fields", func(t *testing.T) {
		record := Record{
			StatusCode: 200,
			Body:       []byte(`{"message":"hello"}`),
			ETag:       `"abc123"`,
		}

		zhtest.AssertEqual(t, 200, record.StatusCode)
		zhtest.AssertEqual(t, `{"message":"hello"}`, string(record.Body))
		zhtest.AssertEqual(t, `"abc123"`, record.ETag)
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

		zhtest.AssertEqual(t, "public, max-age=3600", cfg.CacheControl)
		zhtest.AssertEqual(t, time.Hour, cfg.DefaultTTL)
		zhtest.AssertEqual(t, 5*1024*1024, cfg.MaxBodySize)
		zhtest.AssertEqual(t, 5000, cfg.MaxEntries)
		zhtest.AssertFalse(t, cfg.ETag)
		zhtest.AssertFalse(t, cfg.LastModified)
		zhtest.AssertEqual(t, customStore, cfg.Store)
		zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	})
}

func TestCacheConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
		zhtest.AssertEqual(t, "/health", cfg.IncludedPaths[1])
	})

	t.Run("both excluded and included paths", func(t *testing.T) {
		excludedPaths := []string{"/api/live", "/health"}
		includedPaths := []string{"/api/public"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
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

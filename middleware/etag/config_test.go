package etag

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultETagConfig(t *testing.T) {
	cfg := DefaultConfig

	zhtest.AssertEqual(t, FNV, cfg.Algorithm)
	zhtest.AssertNotNil(t, cfg.Weak)
	zhtest.AssertFalse(t, *cfg.Weak)
	zhtest.AssertEqual(t, 1024*1024, cfg.MaxBufferSize)
	zhtest.AssertNotNil(t, cfg.SkipStatusCodes)
	zhtest.AssertNotNil(t, cfg.SkipContentTypes)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	zhtest.AssertNil(t, cfg.ExcludedFunc)

	// Check some specific skip status codes
	_, ok := cfg.SkipStatusCodes[http.StatusNoContent]
	zhtest.AssertTrue(t, ok)

	_, ok = cfg.SkipStatusCodes[http.StatusInternalServerError]
	zhtest.AssertTrue(t, ok)

	// Check SSE is in skip content types
	_, ok = cfg.SkipContentTypes["text/event-stream"]
	zhtest.AssertTrue(t, ok)
}

func TestETagConfig_StructAssignment(t *testing.T) {
	t.Run("algorithm assignment", func(t *testing.T) {
		cfg := Config{
			Algorithm: MD5,
		}
		zhtest.AssertEqual(t, MD5, cfg.Algorithm)
	})

	t.Run("weak assignment", func(t *testing.T) {
		cfg := Config{
			Weak: config.Bool(false),
		}
		zhtest.AssertNotNil(t, cfg.Weak)
		zhtest.AssertFalse(t, *cfg.Weak)
	})

	t.Run("max buffer size assignment", func(t *testing.T) {
		cfg := Config{
			MaxBufferSize: 512 * 1024,
		}
		zhtest.AssertEqual(t, 512*1024, cfg.MaxBufferSize)
	})

	t.Run("skip status codes assignment", func(t *testing.T) {
		skipCodes := map[int]struct{}{
			http.StatusOK:      {},
			http.StatusCreated: {},
		}
		cfg := Config{
			SkipStatusCodes: skipCodes,
		}
		zhtest.AssertEqual(t, 2, len(cfg.SkipStatusCodes))
		_, ok := cfg.SkipStatusCodes[http.StatusOK]
		zhtest.AssertTrue(t, ok)
		_, ok = cfg.SkipStatusCodes[http.StatusCreated]
		zhtest.AssertTrue(t, ok)
		// Should not have the default codes anymore
		_, ok = cfg.SkipStatusCodes[http.StatusNoContent]
		zhtest.AssertFalse(t, ok)
	})

	t.Run("skip content types assignment", func(t *testing.T) {
		skipTypes := map[string]struct{}{
			"application/pdf": {},
			"video/mp4":       {},
		}
		cfg := Config{
			SkipContentTypes: skipTypes,
		}
		zhtest.AssertEqual(t, 2, len(cfg.SkipContentTypes))
		_, ok := cfg.SkipContentTypes["application/pdf"]
		zhtest.AssertTrue(t, ok)
		_, ok = cfg.SkipContentTypes["video/mp4"]
		zhtest.AssertTrue(t, ok)
		// Should not have the default types anymore
		_, ok = cfg.SkipContentTypes["text/event-stream"]
		zhtest.AssertFalse(t, ok)
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		paths := []string{"/api/stream", "/health"}
		cfg := Config{
			ExcludedPaths: paths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, "/api/stream", cfg.ExcludedPaths[0])
		zhtest.AssertEqual(t, "/health", cfg.ExcludedPaths[1])
	})

	t.Run("included paths assignment", func(t *testing.T) {
		paths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: paths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
		zhtest.AssertEqual(t, "/health", cfg.IncludedPaths[1])
	})

	t.Run("excluded func assignment", func(t *testing.T) {
		fn := func(r *http.Request) bool {
			return r.Header.Get("X-Skip-ETag") == "true"
		}
		cfg := Config{
			ExcludedFunc: fn,
		}
		zhtest.AssertNotNil(t, cfg.ExcludedFunc)
		// Test the function
		req := &http.Request{
			Header: http.Header{"X-Skip-Etag": []string{"true"}},
		}
		zhtest.AssertTrue(t, cfg.ExcludedFunc(req))
		req2 := &http.Request{
			Header: http.Header{},
		}
		zhtest.AssertFalse(t, cfg.ExcludedFunc(req2))
	})
}

func TestETagConfig_MultipleFields(t *testing.T) {
	excludedPaths := []string{"/api/stream"}
	includedPaths := []string{"/api/public"}
	cfg := Config{
		Algorithm:     MD5,
		Weak:          config.Bool(false),
		MaxBufferSize: 2 * 1024 * 1024,
		ExcludedPaths: excludedPaths,
		IncludedPaths: includedPaths,
	}

	zhtest.AssertEqual(t, MD5, cfg.Algorithm)
	zhtest.AssertNotNil(t, cfg.Weak)
	zhtest.AssertFalse(t, *cfg.Weak)
	zhtest.AssertEqual(t, 2*1024*1024, cfg.MaxBufferSize)
	zhtest.AssertEqual(t, 1, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
}

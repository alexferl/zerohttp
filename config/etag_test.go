package config

import (
	"net/http"
	"testing"
)

func TestDefaultETagConfig(t *testing.T) {
	cfg := DefaultETagConfig

	if cfg.Algorithm != FNV {
		t.Errorf("expected default algorithm to be FNV, got %s", cfg.Algorithm)
	}

	if cfg.Weak == nil || !*cfg.Weak {
		t.Error("expected default Weak to be true")
	}

	if cfg.MaxBufferSize != 1024*1024 {
		t.Errorf("expected default MaxBufferSize to be 1MB, got %d", cfg.MaxBufferSize)
	}

	if cfg.SkipStatusCodes == nil {
		t.Error("expected default SkipStatusCodes to be non-nil")
	}

	if cfg.SkipContentTypes == nil {
		t.Error("expected default SkipContentTypes to be non-nil")
	}

	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default ExemptPaths to be empty, got %d", len(cfg.ExemptPaths))
	}

	if cfg.ExemptFunc != nil {
		t.Error("expected default ExemptFunc to be nil")
	}

	// Check some specific skip status codes
	if _, ok := cfg.SkipStatusCodes[http.StatusNoContent]; !ok {
		t.Error("expected 204 No Content to be in SkipStatusCodes")
	}

	if _, ok := cfg.SkipStatusCodes[http.StatusInternalServerError]; !ok {
		t.Error("expected 500 Internal Server Error to be in SkipStatusCodes")
	}

	// Check SSE is in skip content types
	if _, ok := cfg.SkipContentTypes["text/event-stream"]; !ok {
		t.Error("expected text/event-stream to be in SkipContentTypes")
	}
}

func TestETagConfig_StructAssignment(t *testing.T) {
	t.Run("algorithm assignment", func(t *testing.T) {
		cfg := ETagConfig{
			Algorithm: MD5,
		}
		if cfg.Algorithm != MD5 {
			t.Errorf("expected algorithm to be MD5, got %s", cfg.Algorithm)
		}
	})

	t.Run("weak assignment", func(t *testing.T) {
		cfg := ETagConfig{
			Weak: Bool(false),
		}
		if cfg.Weak != nil && *cfg.Weak {
			t.Error("expected Weak to be false")
		}
	})

	t.Run("max buffer size assignment", func(t *testing.T) {
		cfg := ETagConfig{
			MaxBufferSize: 512 * 1024,
		}
		if cfg.MaxBufferSize != 512*1024 {
			t.Errorf("expected MaxBufferSize to be 512KB, got %d", cfg.MaxBufferSize)
		}
	})

	t.Run("skip status codes assignment", func(t *testing.T) {
		skipCodes := map[int]struct{}{
			http.StatusOK:      {},
			http.StatusCreated: {},
		}
		cfg := ETagConfig{
			SkipStatusCodes: skipCodes,
		}
		if len(cfg.SkipStatusCodes) != 2 {
			t.Errorf("expected 2 skip status codes, got %d", len(cfg.SkipStatusCodes))
		}
		if _, ok := cfg.SkipStatusCodes[http.StatusOK]; !ok {
			t.Error("expected 200 OK to be in SkipStatusCodes")
		}
		if _, ok := cfg.SkipStatusCodes[http.StatusCreated]; !ok {
			t.Error("expected 201 Created to be in SkipStatusCodes")
		}
		// Should not have the default codes anymore
		if _, ok := cfg.SkipStatusCodes[http.StatusNoContent]; ok {
			t.Error("expected 204 No Content to NOT be in SkipStatusCodes after override")
		}
	})

	t.Run("skip content types assignment", func(t *testing.T) {
		skipTypes := map[string]struct{}{
			"application/pdf": {},
			"video/mp4":       {},
		}
		cfg := ETagConfig{
			SkipContentTypes: skipTypes,
		}
		if len(cfg.SkipContentTypes) != 2 {
			t.Errorf("expected 2 skip content types, got %d", len(cfg.SkipContentTypes))
		}
		if _, ok := cfg.SkipContentTypes["application/pdf"]; !ok {
			t.Error("expected application/pdf to be in SkipContentTypes")
		}
		if _, ok := cfg.SkipContentTypes["video/mp4"]; !ok {
			t.Error("expected video/mp4 to be in SkipContentTypes")
		}
		// Should not have the default types anymore
		if _, ok := cfg.SkipContentTypes["text/event-stream"]; ok {
			t.Error("expected text/event-stream to NOT be in SkipContentTypes after override")
		}
	})

	t.Run("exempt paths assignment", func(t *testing.T) {
		paths := []string{"/api/stream", "/health"}
		cfg := ETagConfig{
			ExemptPaths: paths,
		}
		if len(cfg.ExemptPaths) != 2 {
			t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if cfg.ExemptPaths[0] != "/api/stream" {
			t.Errorf("expected first path to be /api/stream, got %s", cfg.ExemptPaths[0])
		}
		if cfg.ExemptPaths[1] != "/health" {
			t.Errorf("expected second path to be /health, got %s", cfg.ExemptPaths[1])
		}
	})

	t.Run("exempt func assignment", func(t *testing.T) {
		fn := func(r *http.Request) bool {
			return r.Header.Get("X-Skip-ETag") == "true"
		}
		cfg := ETagConfig{
			ExemptFunc: fn,
		}
		if cfg.ExemptFunc == nil {
			t.Fatal("expected ExemptFunc to be set")
		}
		// Test the function
		req := &http.Request{
			Header: http.Header{"X-Skip-Etag": []string{"true"}},
		}
		if !cfg.ExemptFunc(req) {
			t.Error("expected ExemptFunc to return true for X-Skip-ETag: true")
		}
		req2 := &http.Request{
			Header: http.Header{},
		}
		if cfg.ExemptFunc(req2) {
			t.Error("expected ExemptFunc to return false when X-Skip-ETag is not set")
		}
	})
}

func TestETagConfig_MultipleFields(t *testing.T) {
	cfg := ETagConfig{
		Algorithm:     MD5,
		Weak:          Bool(false),
		MaxBufferSize: 2 * 1024 * 1024,
	}

	if cfg.Algorithm != MD5 {
		t.Errorf("expected algorithm to be MD5, got %s", cfg.Algorithm)
	}

	if cfg.Weak != nil && *cfg.Weak {
		t.Error("expected Weak to be false")
	}

	if cfg.MaxBufferSize != 2*1024*1024 {
		t.Errorf("expected MaxBufferSize to be 2MB, got %d", cfg.MaxBufferSize)
	}
}

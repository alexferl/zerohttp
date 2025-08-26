package config

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestNoCacheConfig_DefaultValues(t *testing.T) {
	cfg := DefaultNoCacheConfig
	if len(cfg.NoCacheHeaders) != 4 {
		t.Errorf("expected 4 default no-cache headers, got %d", len(cfg.NoCacheHeaders))
	}
	if len(cfg.ETagHeaders) != 6 {
		t.Errorf("expected 6 default ETag headers, got %d", len(cfg.ETagHeaders))
	}
	if !reflect.DeepEqual(cfg.NoCacheHeaders, DefaultNoCacheHeaders) {
		t.Errorf("expected default no-cache headers to match DefaultNoCacheHeaders")
	}
	if !reflect.DeepEqual(cfg.ETagHeaders, DefaultETagHeaders) {
		t.Errorf("expected default ETag headers to match DefaultETagHeaders")
	}

	// Test Epoch constant
	expectedEpoch := time.Unix(0, 0).UTC().Format(http.TimeFormat)
	if Epoch != expectedEpoch {
		t.Errorf("expected Epoch = %s, got %s", expectedEpoch, Epoch)
	}
	if DefaultNoCacheHeaders["Expires"] != Epoch {
		t.Errorf("expected DefaultNoCacheHeaders[Expires] to use Epoch constant")
	}

	expectedNoCacheHeaders := map[string]string{
		"Expires":         Epoch,
		"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	if !reflect.DeepEqual(cfg.NoCacheHeaders, expectedNoCacheHeaders) {
		t.Errorf("expected no-cache headers = %v, got %v", expectedNoCacheHeaders, cfg.NoCacheHeaders)
	}

	// Test default ETag headers
	expectedETagHeaders := []string{"ETag", "If-Modified-Since", "If-Match", "If-None-Match", "If-Range", "If-Unmodified-Since"}
	if !reflect.DeepEqual(cfg.ETagHeaders, expectedETagHeaders) {
		t.Errorf("expected ETag headers = %v, got %v", expectedETagHeaders, cfg.ETagHeaders)
	}
}

func TestNoCacheOptions(t *testing.T) {
	t.Run("no-cache headers", func(t *testing.T) {
		customHeaders := map[string]string{
			"Cache-Control": "no-cache, must-revalidate",
			"Expires":       "0",
			"Pragma":        "no-cache",
		}
		cfg := DefaultNoCacheConfig
		WithNoCacheHeaders(customHeaders)(&cfg)
		if len(cfg.NoCacheHeaders) != 3 {
			t.Errorf("expected 3 no-cache headers, got %d", len(cfg.NoCacheHeaders))
		}
		if !reflect.DeepEqual(cfg.NoCacheHeaders, customHeaders) {
			t.Errorf("expected no-cache headers = %v, got %v", customHeaders, cfg.NoCacheHeaders)
		}
	})

	t.Run("etag headers", func(t *testing.T) {
		customETagHeaders := []string{"ETag", "If-None-Match", "Last-Modified"}
		cfg := DefaultNoCacheConfig
		WithNoCacheETagHeaders(customETagHeaders)(&cfg)
		if len(cfg.ETagHeaders) != 3 {
			t.Errorf("expected 3 ETag headers, got %d", len(cfg.ETagHeaders))
		}
		if !reflect.DeepEqual(cfg.ETagHeaders, customETagHeaders) {
			t.Errorf("expected ETag headers = %v, got %v", customETagHeaders, cfg.ETagHeaders)
		}
	})

	t.Run("additional headers", func(t *testing.T) {
		customHeaders := map[string]string{
			"Cache-Control":     "no-cache",
			"Expires":           Epoch,
			"Pragma":            "no-cache",
			"X-Accel-Expires":   "0",
			"Surrogate-Control": "no-store",
			"Vary":              "*",
			"Last-Modified":     Epoch,
		}
		cfg := DefaultNoCacheConfig
		WithNoCacheHeaders(customHeaders)(&cfg)
		if len(cfg.NoCacheHeaders) != 7 {
			t.Errorf("expected 7 headers, got %d", len(cfg.NoCacheHeaders))
		}
		for key, expectedValue := range customHeaders {
			if cfg.NoCacheHeaders[key] != expectedValue {
				t.Errorf("expected header %s = %s, got %s", key, expectedValue, cfg.NoCacheHeaders[key])
			}
		}
	})

	t.Run("all etag related headers", func(t *testing.T) {
		allETagHeaders := []string{"ETag", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "If-Range", "Last-Modified", "Vary"}
		cfg := DefaultNoCacheConfig
		WithNoCacheETagHeaders(allETagHeaders)(&cfg)
		if len(cfg.ETagHeaders) != 8 {
			t.Errorf("expected 8 ETag-related headers, got %d", len(cfg.ETagHeaders))
		}
		if !reflect.DeepEqual(cfg.ETagHeaders, allETagHeaders) {
			t.Errorf("expected ETag headers = %v, got %v", allETagHeaders, cfg.ETagHeaders)
		}
	})
}

func TestNoCacheConfig_MultipleOptions(t *testing.T) {
	customNoCacheHeaders := map[string]string{
		"Cache-Control": "no-store",
		"Expires":       "-1",
	}
	customETagHeaders := []string{"ETag", "If-Match"}
	cfg := DefaultNoCacheConfig
	WithNoCacheHeaders(customNoCacheHeaders)(&cfg)
	WithNoCacheETagHeaders(customETagHeaders)(&cfg)

	if !reflect.DeepEqual(cfg.NoCacheHeaders, customNoCacheHeaders) {
		t.Error("expected no-cache headers to be set correctly")
	}
	if !reflect.DeepEqual(cfg.ETagHeaders, customETagHeaders) {
		t.Error("expected ETag headers to be set correctly")
	}
	if len(cfg.NoCacheHeaders) != 2 {
		t.Errorf("expected 2 no-cache headers, got %d", len(cfg.NoCacheHeaders))
	}
	if len(cfg.ETagHeaders) != 2 {
		t.Errorf("expected 2 ETag headers, got %d", len(cfg.ETagHeaders))
	}
}

func TestNoCacheConfig_EdgeCases(t *testing.T) {
	t.Run("empty headers", func(t *testing.T) {
		cfg := DefaultNoCacheConfig
		WithNoCacheHeaders(map[string]string{})(&cfg)
		WithNoCacheETagHeaders([]string{})(&cfg)

		if cfg.NoCacheHeaders == nil || len(cfg.NoCacheHeaders) != 0 {
			t.Errorf("expected empty no-cache headers map, got %v", cfg.NoCacheHeaders)
		}
		if cfg.ETagHeaders == nil || len(cfg.ETagHeaders) != 0 {
			t.Errorf("expected empty ETag headers slice, got %v", cfg.ETagHeaders)
		}
	})

	t.Run("nil headers", func(t *testing.T) {
		cfg := DefaultNoCacheConfig
		WithNoCacheHeaders(nil)(&cfg)
		WithNoCacheETagHeaders(nil)(&cfg)

		if cfg.NoCacheHeaders != nil {
			t.Error("expected no-cache headers to remain nil when nil is passed")
		}
		if cfg.ETagHeaders != nil {
			t.Error("expected ETag headers to remain nil when nil is passed")
		}
	})

	t.Run("case sensitive headers", func(t *testing.T) {
		headers := map[string]string{
			"cache-control": "no-cache",
			"Cache-Control": "no-store",
			"CACHE-CONTROL": "must-revalidate",
		}
		cfg := DefaultNoCacheConfig
		WithNoCacheHeaders(headers)(&cfg)
		if len(cfg.NoCacheHeaders) != 3 {
			t.Errorf("expected 3 headers (case-sensitive keys), got %d", len(cfg.NoCacheHeaders))
		}
		if cfg.NoCacheHeaders["cache-control"] != "no-cache" {
			t.Error("expected lowercase cache-control to be preserved")
		}
		if cfg.NoCacheHeaders["Cache-Control"] != "no-store" {
			t.Error("expected title-case Cache-Control to be preserved")
		}
		if cfg.NoCacheHeaders["CACHE-CONTROL"] != "must-revalidate" {
			t.Error("expected uppercase CACHE-CONTROL to be preserved")
		}
	})

	t.Run("etag header variations", func(t *testing.T) {
		etagHeaders := []string{"ETag", "etag", "ETAG", "If-Modified-Since", "if-modified-since", "IF-MODIFIED-SINCE", "If-None-Match", "Last-Modified", "last-modified"}
		cfg := DefaultNoCacheConfig
		WithNoCacheETagHeaders(etagHeaders)(&cfg)
		if len(cfg.ETagHeaders) != 9 {
			t.Errorf("expected 9 ETag headers, got %d", len(cfg.ETagHeaders))
		}
		if !reflect.DeepEqual(cfg.ETagHeaders, etagHeaders) {
			t.Errorf("expected ETag headers = %v, got %v", etagHeaders, cfg.ETagHeaders)
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		headers := map[string]string{
			"Cache-Control": "",
			"Expires":       "",
			"Pragma":        "",
		}
		etagHeaders := []string{"", "ETag", ""}
		cfg := DefaultNoCacheConfig
		WithNoCacheHeaders(headers)(&cfg)
		WithNoCacheETagHeaders(etagHeaders)(&cfg)

		if len(cfg.NoCacheHeaders) != 3 {
			t.Errorf("expected 3 no-cache headers, got %d", len(cfg.NoCacheHeaders))
		}
		if len(cfg.ETagHeaders) != 3 {
			t.Errorf("expected 3 ETag headers, got %d", len(cfg.ETagHeaders))
		}

		for key, value := range headers {
			if cfg.NoCacheHeaders[key] != value {
				t.Errorf("expected header %s = %q, got %q", key, value, cfg.NoCacheHeaders[key])
			}
		}
		for i, expectedHeader := range etagHeaders {
			if cfg.ETagHeaders[i] != expectedHeader {
				t.Errorf("expected ETag header[%d] = %q, got %q", i, expectedHeader, cfg.ETagHeaders[i])
			}
		}
	})
}

func TestNoCacheConfig_CacheControlDirectives(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl string
	}{
		{"strict no-cache", "no-cache, no-store, must-revalidate"},
		{"with max-age", "no-cache, max-age=0"},
		{"with private", "no-cache, private"},
		{"with no-transform", "no-cache, no-transform"},
		{"minimal", "no-cache"},
		{"comprehensive", "no-cache, no-store, no-transform, must-revalidate, private, max-age=0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Cache-Control": tt.cacheControl}
			cfg := DefaultNoCacheConfig
			WithNoCacheHeaders(headers)(&cfg)
			if cfg.NoCacheHeaders["Cache-Control"] != tt.cacheControl {
				t.Errorf("expected Cache-Control = %s, got %s", tt.cacheControl, cfg.NoCacheHeaders["Cache-Control"])
			}
		})
	}
}

func TestNoCacheConfig_ExpiresValues(t *testing.T) {
	tests := []struct {
		name    string
		expires string
	}{
		{"epoch", Epoch},
		{"zero", "0"},
		{"negative", "-1"},
		{"past date", "Thu, 01 Jan 1970 00:00:00 GMT"},
		{"far past", "Mon, 01 Jan 1900 00:00:00 GMT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Expires": tt.expires}
			cfg := DefaultNoCacheConfig
			WithNoCacheHeaders(headers)(&cfg)
			if cfg.NoCacheHeaders["Expires"] != tt.expires {
				t.Errorf("expected Expires = %s, got %s", tt.expires, cfg.NoCacheHeaders["Expires"])
			}
		})
	}
}

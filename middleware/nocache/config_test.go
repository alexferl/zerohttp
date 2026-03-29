package nocache

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestNoCacheConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 4, len(cfg.Headers))
	zhtest.AssertEqual(t, 6, len(cfg.ETagHeaders))
	zhtest.AssertEqual(t, DefaultHeaders, cfg.Headers)
	zhtest.AssertEqual(t, DefaultETagHeaders, cfg.ETagHeaders)

	// Test Epoch constant
	expectedEpoch := time.Unix(0, 0).UTC().Format(http.TimeFormat)
	zhtest.AssertEqual(t, expectedEpoch, Epoch)
	zhtest.AssertEqual(t, Epoch, DefaultHeaders["Expires"])

	expectedNoCacheHeaders := map[string]string{
		"Expires":         Epoch,
		"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	zhtest.AssertEqual(t, expectedNoCacheHeaders, cfg.Headers)

	// Test default ETag headers
	expectedETagHeaders := []string{"ETag", "If-Modified-Since", "If-Match", "If-None-Match", "If-Range", "If-Unmodified-Since"}
	zhtest.AssertEqual(t, expectedETagHeaders, cfg.ETagHeaders)
}

func TestNoCacheConfig_StructAssignment(t *testing.T) {
	t.Run("no-cache headers", func(t *testing.T) {
		customHeaders := map[string]string{
			"Cache-Control": "no-cache, must-revalidate",
			"Expires":       "0",
			"Pragma":        "no-cache",
		}
		cfg := Config{
			Headers: customHeaders,
		}
		zhtest.AssertEqual(t, 3, len(cfg.Headers))
		zhtest.AssertEqual(t, customHeaders, cfg.Headers)
	})

	t.Run("etag headers", func(t *testing.T) {
		customETagHeaders := []string{"ETag", "If-None-Match", "Last-Modified"}
		cfg := Config{
			ETagHeaders: customETagHeaders,
		}
		zhtest.AssertEqual(t, 3, len(cfg.ETagHeaders))
		zhtest.AssertEqual(t, customETagHeaders, cfg.ETagHeaders)
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
		cfg := Config{
			Headers: customHeaders,
		}
		zhtest.AssertEqual(t, 7, len(cfg.Headers))
		for key, expectedValue := range customHeaders {
			zhtest.AssertEqual(t, expectedValue, cfg.Headers[key])
		}
	})

	t.Run("all etag related headers", func(t *testing.T) {
		allETagHeaders := []string{"ETag", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "If-Range", "Last-Modified", "Vary"}
		cfg := Config{
			ETagHeaders: allETagHeaders,
		}
		zhtest.AssertEqual(t, 8, len(cfg.ETagHeaders))
		zhtest.AssertEqual(t, allETagHeaders, cfg.ETagHeaders)
	})
}

func TestNoCacheConfig_MultipleFields(t *testing.T) {
	customNoCacheHeaders := map[string]string{
		"Cache-Control": "no-store",
		"Expires":       "-1",
	}
	customETagHeaders := []string{"ETag", "If-Match"}
	cfg := Config{
		Headers:     customNoCacheHeaders,
		ETagHeaders: customETagHeaders,
	}

	zhtest.AssertEqual(t, customNoCacheHeaders, cfg.Headers)
	zhtest.AssertEqual(t, customETagHeaders, cfg.ETagHeaders)
	zhtest.AssertEqual(t, 2, len(cfg.Headers))
	zhtest.AssertEqual(t, 2, len(cfg.ETagHeaders))
}

func TestNoCacheConfig_EdgeCases(t *testing.T) {
	t.Run("empty headers", func(t *testing.T) {
		cfg := Config{
			Headers:     map[string]string{},
			ETagHeaders: []string{},
		}

		zhtest.AssertNotNil(t, cfg.Headers)
		zhtest.AssertEqual(t, 0, len(cfg.Headers))
		zhtest.AssertNotNil(t, cfg.ETagHeaders)
		zhtest.AssertEqual(t, 0, len(cfg.ETagHeaders))
	})

	t.Run("nil headers", func(t *testing.T) {
		cfg := Config{
			Headers:     nil,
			ETagHeaders: nil,
		}

		zhtest.AssertNil(t, cfg.Headers)
		zhtest.AssertNil(t, cfg.ETagHeaders)
	})

	t.Run("case sensitive headers", func(t *testing.T) {
		headers := map[string]string{
			"cache-control": "no-cache",
			"Cache-Control": "no-store",
			"CACHE-CONTROL": "must-revalidate",
		}
		cfg := Config{
			Headers: headers,
		}
		zhtest.AssertEqual(t, 3, len(cfg.Headers))
		zhtest.AssertEqual(t, "no-cache", cfg.Headers["cache-control"])
		zhtest.AssertEqual(t, "no-store", cfg.Headers["Cache-Control"])
		zhtest.AssertEqual(t, "must-revalidate", cfg.Headers["CACHE-CONTROL"])
	})

	t.Run("etag header variations", func(t *testing.T) {
		etagHeaders := []string{"ETag", "etag", "ETAG", "If-Modified-Since", "if-modified-since", "IF-MODIFIED-SINCE", "If-None-Match", "Last-Modified", "last-modified"}
		cfg := Config{
			ETagHeaders: etagHeaders,
		}
		zhtest.AssertEqual(t, 9, len(cfg.ETagHeaders))
		zhtest.AssertEqual(t, etagHeaders, cfg.ETagHeaders)
	})

	t.Run("empty string values", func(t *testing.T) {
		headers := map[string]string{
			"Cache-Control": "",
			"Expires":       "",
			"Pragma":        "",
		}
		etagHeaders := []string{"", "ETag", ""}
		cfg := Config{
			Headers:     headers,
			ETagHeaders: etagHeaders,
		}

		zhtest.AssertEqual(t, 3, len(cfg.Headers))
		zhtest.AssertEqual(t, 3, len(cfg.ETagHeaders))

		for key, value := range headers {
			zhtest.AssertEqual(t, value, cfg.Headers[key])
		}
		for i, expectedHeader := range etagHeaders {
			zhtest.AssertEqual(t, expectedHeader, cfg.ETagHeaders[i])
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
			cfg := Config{
				Headers: headers,
			}
			zhtest.AssertEqual(t, tt.cacheControl, cfg.Headers["Cache-Control"])
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
			cfg := Config{
				Headers: headers,
			}
			zhtest.AssertEqual(t, tt.expires, cfg.Headers["Expires"])
		})
	}
}

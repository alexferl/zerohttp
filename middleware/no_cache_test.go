package middleware

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestNoCacheEpochValue(t *testing.T) {
	expectedEpoch := time.Unix(0, 0).UTC().Format(http.TimeFormat)
	if config.Epoch != expectedEpoch {
		t.Errorf("expected config.Epoch %s, got %s", expectedEpoch, config.Epoch)
	}
	if expectedEpoch != "Thu, 01 Jan 1970 00:00:00 GMT" {
		t.Errorf("expected config.Epoch to be Thu, 01 Jan 1970 00:00:00 GMT, got %s", expectedEpoch)
	}
}

func TestNoCacheResponseHeaders(t *testing.T) {
	middleware := NoCache()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware(next), req)

	expectedHeaders := map[string]string{
		"Expires":         config.Epoch,
		"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	for header, expected := range expectedHeaders {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("expected %s header = %s, got %s", header, expected, got)
		}
	}
	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("test response")
}

func TestNoCacheRequestHeaderRemoval(t *testing.T) {
	middleware := NoCache()
	headers := map[string]string{
		"ETag":                `"abc123"`,
		"If-Modified-Since":   "Wed, 21 Oct 2015 07:28:00 GMT",
		"If-Match":            `"abc123"`,
		"If-None-Match":       `"def456"`,
		"If-Range":            `"ghi789"`,
		"If-Unmodified-Since": "Wed, 21 Oct 2015 07:28:00 GMT",
		"User-Agent":          "test-agent",
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range []string{"ETag", "If-Modified-Since", "If-Match", "If-None-Match", "If-Range", "If-Unmodified-Since"} {
			if value := r.Header.Get(header); value != "" {
				t.Errorf("expected %s header to be removed, but got %s", header, value)
			}
		}
		if userAgent := r.Header.Get("User-Agent"); userAgent != "test-agent" {
			t.Errorf("expected User-Agent to remain, got %s", userAgent)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeaders(headers).Build()
	zhtest.Serve(middleware(next), req)
}

func TestNoCacheCustomConfig(t *testing.T) {
	middleware := NoCache(config.NoCacheConfig{
		NoCacheHeaders: map[string]string{
			"Cache-Control": "no-cache",
			"Expires":       "0",
		},
		ETagHeaders: []string{"ETag", "If-None-Match"},
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get("ETag"); etag != "" {
			t.Errorf("expected ETag to be removed, got %s", etag)
		}
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
			t.Errorf("expected If-None-Match to be removed, got %s", ifNoneMatch)
		}
		if ifModified := r.Header.Get("If-Modified-Since"); ifModified == "" {
			t.Error("expected If-Modified-Since to remain")
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").
		WithHeader("ETag", `"test123"`).
		WithHeader("If-None-Match", `"test456"`).
		WithHeader("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT").
		Build()
	w := zhtest.Serve(middleware(next), req)

	zhtest.AssertWith(t, w).Header("Cache-Control", "no-cache").Header("Expires", "0").HeaderNotExists("Pragma")
}

func TestNoCacheNilConfig(t *testing.T) {
	middleware := NoCache(config.NoCacheConfig{
		NoCacheHeaders: nil,
		ETagHeaders:    nil,
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get("ETag"); etag != "" {
			t.Errorf("expected ETag to be removed with nil config, got %s", etag)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("ETag", `"test123"`).Build()
	w := zhtest.Serve(middleware(next), req)

	for _, header := range []string{"Expires", "Cache-Control", "Pragma", "X-Accel-Expires"} {
		zhtest.AssertWith(t, w).HeaderExists(header)
	}
}

func TestNoCacheHTTPMethods(t *testing.T) {
	middleware := NoCache()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if etag := r.Header.Get("ETag"); etag != "" {
					t.Errorf("expected ETag to be removed for %s method, got %s", method, etag)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := zhtest.NewRequest(method, "/test").WithHeader("ETag", `"test123"`).Build()
			w := zhtest.Serve(middleware(next), req)

			zhtest.AssertWith(t, w).HeaderExists("Cache-Control").Status(http.StatusOK)
		})
	}
}

func TestNoCacheEmptyHeaders(t *testing.T) {
	middleware := NoCache(config.NoCacheConfig{
		NoCacheHeaders: map[string]string{},
		ETagHeaders:    []string{},
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get("ETag"); etag != `"test123"` {
			t.Errorf("expected ETag to remain with empty config, got %s", etag)
		}
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != `"test456"` {
			t.Errorf("expected If-None-Match to remain with empty config, got %s", ifNoneMatch)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").
		WithHeader("ETag", `"test123"`).
		WithHeader("If-None-Match", `"test456"`).
		Build()
	w := zhtest.Serve(middleware(next), req)

	for _, header := range []string{"Expires", "Cache-Control", "Pragma", "X-Accel-Expires"} {
		zhtest.AssertWith(t, w).HeaderNotExists(header)
	}
}

func TestNoCacheOnlyExistingHeaders(t *testing.T) {
	middleware := NoCache()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" {
			t.Errorf("expected If-Match to be removed, got %s", ifMatch)
		}
		if userAgent := r.Header.Get("User-Agent"); userAgent != "test-agent" {
			t.Errorf("expected User-Agent to remain, got %s", userAgent)
		}
		if etag := r.Header.Get("ETag"); etag != "" {
			t.Errorf("expected ETag to remain empty, got %s", etag)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").
		WithHeader("If-Match", `"abc123"`).
		WithHeader("User-Agent", "test-agent").
		Build()
	zhtest.Serve(middleware(next), req)
}

func TestNoCacheResponseAndRequest(t *testing.T) {
	middleware := NoCache()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
			t.Errorf("expected If-None-Match to be removed, got %s", ifNoneMatch)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token123" {
			t.Errorf("expected Authorization to remain, got %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": "test"}`))
	})

	req := zhtest.NewRequest(http.MethodGet, "/api/data").
		WithHeader("If-None-Match", `"cached-etag"`).
		WithHeader("Authorization", "Bearer token123").
		Build()
	w := zhtest.Serve(middleware(next), req)

	if cacheControl := w.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("expected Cache-Control to contain no-cache, got %s", cacheControl)
	}
	zhtest.AssertWith(t, w).
		Header("Content-Type", "application/json").
		Body(`{"data": "test"}`)
}

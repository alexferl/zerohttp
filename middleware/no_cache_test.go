package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
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
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	middleware(next).ServeHTTP(rr, req)
	expectedHeaders := map[string]string{
		"Expires":         config.Epoch,
		"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	for header, expected := range expectedHeaders {
		if got := rr.Header().Get(header); got != expected {
			t.Errorf("expected %s header = %s, got %s", header, expected, got)
		}
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "test response" {
		t.Errorf("expected body 'test response', got %s", body)
	}
}

func TestNoCacheRequestHeaderRemoval(t *testing.T) {
	middleware := NoCache()
	req := httptest.NewRequest("GET", "/test", nil)
	headers := map[string]string{
		"ETag":                "\"abc123\"",
		"If-Modified-Since":   "Wed, 21 Oct 2015 07:28:00 GMT",
		"If-Match":            "\"abc123\"",
		"If-None-Match":       "\"def456\"",
		"If-Range":            "\"ghi789\"",
		"If-Unmodified-Since": "Wed, 21 Oct 2015 07:28:00 GMT",
		"User-Agent":          "test-agent",
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
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
	middleware(next).ServeHTTP(rr, req)
}

func TestNoCacheCustomConfig(t *testing.T) {
	middleware := NoCache(
		config.WithNoCacheHeaders(map[string]string{
			"Cache-Control": "no-cache",
			"Expires":       "0",
		}),
		config.WithNoCacheETagHeaders([]string{"ETag", "If-None-Match"}),
	)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("ETag", "\"test123\"")
	req.Header.Set("If-None-Match", "\"test456\"")
	req.Header.Set("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT")
	rr := httptest.NewRecorder()
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
	middleware(next).ServeHTTP(rr, req)

	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %s", cacheControl)
	}
	if expires := rr.Header().Get("Expires"); expires != "0" {
		t.Errorf("expected Expires '0', got %s", expires)
	}
	if pragma := rr.Header().Get("Pragma"); pragma != "" {
		t.Errorf("expected Pragma to be empty with custom config, got %s", pragma)
	}
}

func TestNoCacheNilConfig(t *testing.T) {
	middleware := NoCache(
		config.WithNoCacheHeaders(nil),
		config.WithNoCacheETagHeaders(nil),
	)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("ETag", "\"test123\"")
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get("ETag"); etag != "" {
			t.Errorf("expected ETag to be removed with nil config, got %s", etag)
		}
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)

	for _, header := range []string{"Expires", "Cache-Control", "Pragma", "X-Accel-Expires"} {
		if value := rr.Header().Get(header); value == "" {
			t.Errorf("expected %s header to be set with nil config", header)
		}
	}
}

func TestNoCacheHTTPMethods(t *testing.T) {
	middleware := NoCache()
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("ETag", "\"test123\"")
			rr := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if etag := r.Header.Get("ETag"); etag != "" {
					t.Errorf("expected ETag to be removed for %s method, got %s", method, etag)
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware(next).ServeHTTP(rr, req)
			if cacheControl := rr.Header().Get("Cache-Control"); cacheControl == "" {
				t.Errorf("expected Cache-Control header for %s method", method)
			}
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200 for %s method, got %d", method, rr.Code)
			}
		})
	}
}

func TestNoCacheEmptyHeaders(t *testing.T) {
	middleware := NoCache(
		config.WithNoCacheHeaders(map[string]string{}),
		config.WithNoCacheETagHeaders([]string{}),
	)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("ETag", "\"test123\"")
	req.Header.Set("If-None-Match", "\"test456\"")
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get("ETag"); etag != "\"test123\"" {
			t.Errorf("expected ETag to remain with empty config, got %s", etag)
		}
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "\"test456\"" {
			t.Errorf("expected If-None-Match to remain with empty config, got %s", ifNoneMatch)
		}
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)

	for _, header := range []string{"Expires", "Cache-Control", "Pragma", "X-Accel-Expires"} {
		if value := rr.Header().Get(header); value != "" {
			t.Errorf("expected %s header to be empty with empty config, got %s", header, value)
		}
	}
}

func TestNoCacheOnlyExistingHeaders(t *testing.T) {
	middleware := NoCache()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("If-Match", "\"abc123\"")
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()
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
	middleware(next).ServeHTTP(rr, req)
}

func TestNoCacheResponseAndRequest(t *testing.T) {
	middleware := NoCache()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("If-None-Match", "\"cached-etag\"")
	req.Header.Set("Authorization", "Bearer token123")
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
			t.Errorf("expected If-None-Match to be removed, got %s", ifNoneMatch)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token123" {
			t.Errorf("expected Authorization to remain, got %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"data": "test"}`))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	middleware(next).ServeHTTP(rr, req)

	if cacheControl := rr.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("expected Cache-Control to contain no-cache, got %s", cacheControl)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
	if body := rr.Body.String(); body != `{"data": "test"}` {
		t.Errorf("expected JSON body, got %s", body)
	}
}

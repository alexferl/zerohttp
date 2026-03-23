package nocache

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestNoCacheEpochValue(t *testing.T) {
	expectedEpoch := time.Unix(0, 0).UTC().Format(http.TimeFormat)
	if Epoch != expectedEpoch {
		t.Errorf("expected config.Epoch %s, got %s", expectedEpoch, Epoch)
	}
	if expectedEpoch != "Thu, 01 Jan 1970 00:00:00 GMT" {
		t.Errorf("expected config.Epoch to be Thu, 01 Jan 1970 00:00:00 GMT, got %s", expectedEpoch)
	}
}

func TestNoCacheResponseHeaders(t *testing.T) {
	middleware := New()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	w := zhtest.Serve(middleware(next), req)

	expectedHeaders := map[string]string{
		httpx.HeaderExpires:       Epoch,
		httpx.HeaderCacheControl:  "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
		httpx.HeaderPragma:        "no-cache",
		httpx.HeaderXAccelExpires: "0",
	}
	for header, expected := range expectedHeaders {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("expected %s header = %s, got %s", header, expected, got)
		}
	}
	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("test response")
}

func TestNoCacheRequestHeaderRemoval(t *testing.T) {
	middleware := New()
	headers := map[string]string{
		httpx.HeaderETag:              `"abc123"`,
		httpx.HeaderIfModifiedSince:   "Wed, 21 Oct 2015 07:28:00 GMT",
		httpx.HeaderIfMatch:           `"abc123"`,
		httpx.HeaderIfNoneMatch:       `"def456"`,
		httpx.HeaderIfRange:           `"ghi789"`,
		httpx.HeaderIfUnmodifiedSince: "Wed, 21 Oct 2015 07:28:00 GMT",
		httpx.HeaderUserAgent:         "test-agent",
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, header := range []string{httpx.HeaderETag, httpx.HeaderIfModifiedSince, httpx.HeaderIfMatch, httpx.HeaderIfNoneMatch, httpx.HeaderIfRange, httpx.HeaderIfUnmodifiedSince} {
			if value := r.Header.Get(header); value != "" {
				t.Errorf("expected %s header to be removed, but got %s", header, value)
			}
		}
		if userAgent := r.Header.Get(httpx.HeaderUserAgent); userAgent != "test-agent" {
			t.Errorf("expected User-Agent to remain, got %s", userAgent)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeaders(headers).Build()
	zhtest.Serve(middleware(next), req)
}

func TestNoCacheCustomConfig(t *testing.T) {
	middleware := New(Config{
		Headers: map[string]string{
			httpx.HeaderCacheControl: httpx.CacheControlNoCache,
			httpx.HeaderExpires:      "0",
		},
		ETagHeaders: []string{httpx.HeaderETag, httpx.HeaderIfNoneMatch},
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get(httpx.HeaderETag); etag != "" {
			t.Errorf("expected ETag to be removed, got %s", etag)
		}
		if ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch); ifNoneMatch != "" {
			t.Errorf("expected If-None-Match to be removed, got %s", ifNoneMatch)
		}
		if ifModified := r.Header.Get(httpx.HeaderIfModifiedSince); ifModified == "" {
			t.Error("expected If-Modified-Since to remain")
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").
		WithHeader(httpx.HeaderETag, `"test123"`).
		WithHeader(httpx.HeaderIfNoneMatch, `"test456"`).
		WithHeader(httpx.HeaderIfModifiedSince, "Wed, 21 Oct 2015 07:28:00 GMT").
		Build()
	w := zhtest.Serve(middleware(next), req)

	zhtest.AssertWith(t, w).
		Header(httpx.HeaderCacheControl, httpx.CacheControlNoCache).
		Header(httpx.HeaderExpires, "0").
		HeaderNotExists(httpx.HeaderPragma)
}

func TestNoCacheNilConfig(t *testing.T) {
	middleware := New(Config{
		Headers:     nil,
		ETagHeaders: nil,
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get(httpx.HeaderETag); etag != "" {
			t.Errorf("expected ETag to be removed with nil config, got %s", etag)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader(httpx.HeaderETag, `"test123"`).Build()
	w := zhtest.Serve(middleware(next), req)

	for _, header := range []string{httpx.HeaderExpires, httpx.HeaderCacheControl, httpx.HeaderPragma, httpx.HeaderXAccelExpires} {
		zhtest.AssertWith(t, w).HeaderExists(header)
	}
}

func TestNoCacheHTTPMethods(t *testing.T) {
	middleware := New()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if etag := r.Header.Get(httpx.HeaderETag); etag != "" {
					t.Errorf("expected ETag to be removed for %s method, got %s", method, etag)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := zhtest.NewRequest(method, "/test").WithHeader(httpx.HeaderETag, `"test123"`).Build()
			w := zhtest.Serve(middleware(next), req)

			zhtest.AssertWith(t, w).HeaderExists(httpx.HeaderCacheControl).Status(http.StatusOK)
		})
	}
}

func TestNoCacheEmptyHeaders(t *testing.T) {
	middleware := New(Config{
		Headers:     map[string]string{},
		ETagHeaders: []string{},
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag := r.Header.Get(httpx.HeaderETag); etag != `"test123"` {
			t.Errorf("expected ETag to remain with empty config, got %s", etag)
		}
		if ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch); ifNoneMatch != `"test456"` {
			t.Errorf("expected If-None-Match to remain with empty config, got %s", ifNoneMatch)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").
		WithHeader(httpx.HeaderETag, `"test123"`).
		WithHeader(httpx.HeaderIfNoneMatch, `"test456"`).
		Build()
	w := zhtest.Serve(middleware(next), req)

	for _, header := range []string{httpx.HeaderExpires, httpx.HeaderCacheControl, httpx.HeaderPragma, httpx.HeaderXAccelExpires} {
		zhtest.AssertWith(t, w).HeaderNotExists(header)
	}
}

func TestNoCacheOnlyExistingHeaders(t *testing.T) {
	middleware := New()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ifMatch := r.Header.Get(httpx.HeaderIfMatch); ifMatch != "" {
			t.Errorf("expected If-Match to be removed, got %s", ifMatch)
		}
		if userAgent := r.Header.Get(httpx.HeaderUserAgent); userAgent != "test-agent" {
			t.Errorf("expected User-Agent to remain, got %s", userAgent)
		}
		if etag := r.Header.Get(httpx.HeaderETag); etag != "" {
			t.Errorf("expected ETag to remain empty, got %s", etag)
		}
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/test").
		WithHeader(httpx.HeaderIfMatch, `"abc123"`).
		WithHeader(httpx.HeaderUserAgent, "test-agent").
		Build()
	zhtest.Serve(middleware(next), req)
}

func TestNoCacheResponseAndRequest(t *testing.T) {
	middleware := New()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ifNoneMatch := r.Header.Get(httpx.HeaderIfNoneMatch); ifNoneMatch != "" {
			t.Errorf("expected If-None-Match to be removed, got %s", ifNoneMatch)
		}
		if auth := r.Header.Get(httpx.HeaderAuthorization); auth != "Bearer token123" {
			t.Errorf("expected Authorization to remain, got %s", auth)
		}

		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": "test"}`))
	})

	req := zhtest.NewRequest(http.MethodGet, "/api/data").
		WithHeader(httpx.HeaderIfNoneMatch, `"cached-etag"`).
		WithHeader(httpx.HeaderAuthorization, "Bearer token123").
		Build()
	w := zhtest.Serve(middleware(next), req)

	if cacheControl := w.Header().Get(httpx.HeaderCacheControl); !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("expected Cache-Control to contain no-cache, got %s", cacheControl)
	}
	zhtest.AssertWith(t, w).
		Header(httpx.HeaderContentType, "application/json").
		Body(`{"data": "test"}`)
}

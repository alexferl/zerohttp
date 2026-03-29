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
	zhtest.AssertEqual(t, expectedEpoch, Epoch)
	zhtest.AssertEqual(t, "Thu, 01 Jan 1970 00:00:00 GMT", expectedEpoch)
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
		zhtest.AssertEqual(t, expected, w.Header().Get(header))
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
			zhtest.AssertEmpty(t, r.Header.Get(header))
		}
		zhtest.AssertEqual(t, "test-agent", r.Header.Get(httpx.HeaderUserAgent))
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
		zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderETag))
		zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderIfNoneMatch))
		zhtest.AssertNotEmpty(t, r.Header.Get(httpx.HeaderIfModifiedSince))
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
		zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderETag))
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
				zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderETag))
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
		zhtest.AssertEqual(t, `"test123"`, r.Header.Get(httpx.HeaderETag))
		zhtest.AssertEqual(t, `"test456"`, r.Header.Get(httpx.HeaderIfNoneMatch))
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
		zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderIfMatch))
		zhtest.AssertEqual(t, "test-agent", r.Header.Get(httpx.HeaderUserAgent))
		zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderETag))
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
		zhtest.AssertEmpty(t, r.Header.Get(httpx.HeaderIfNoneMatch))
		zhtest.AssertEqual(t, "Bearer token123", r.Header.Get(httpx.HeaderAuthorization))

		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": "test"}`))
	})

	req := zhtest.NewRequest(http.MethodGet, "/api/data").
		WithHeader(httpx.HeaderIfNoneMatch, `"cached-etag"`).
		WithHeader(httpx.HeaderAuthorization, "Bearer token123").
		Build()
	w := zhtest.Serve(middleware(next), req)

	zhtest.AssertTrue(t, strings.Contains(w.Header().Get(httpx.HeaderCacheControl), "no-cache"))
	zhtest.AssertWith(t, w).
		Header(httpx.HeaderContentType, "application/json").
		Body(`{"data": "test"}`)
}

package mediatype

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestMediaTypeAcceptValidation(t *testing.T) {
	tests := []struct {
		name       string
		accept     string
		expectNext bool
		expectCode int
	}{
		{"allowed exact type", "application/vnd.api+json", true, http.StatusOK},
		{"allowed wildcard suffix", "application/custom+json", true, http.StatusOK},
		{"disallowed type", "text/plain", false, http.StatusNotAcceptable},
		{"disallowed xml", "application/xml", false, http.StatusNotAcceptable},
		{"*/* allowed", "*/*", true, http.StatusOK},
		{"no accept header", "", true, http.StatusOK},
		{"multiple with valid", "text/plain, application/vnd.api+json", true, http.StatusOK},
		{"multiple invalid", "text/plain, application/xml", false, http.StatusNotAcceptable},
		{"with q-value", "application/vnd.api+json;q=0.9", true, http.StatusOK},
		{"case insensitive", "APPLICATION/VND.API+JSON", true, http.StatusOK},
		{"whitespace trimmed", " application/vnd.api+json ", true, http.StatusOK},
		{"malformed accept header rejected", ";;;", false, http.StatusNotAcceptable},
		{"q-value ignored", "application/vnd.api+json;q=0", true, http.StatusOK},
		{"accept with charset", "application/vnd.api+json; charset=utf-8", true, http.StatusOK},
	}

	middleware := New(Config{AllowedTypes: []string{"application/vnd.api+json", "application/*+json"}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.accept != "" {
				req.Header.Set(httpx.HeaderAccept, tt.accept)
			}

			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			zhtest.AssertEqual(t, tt.expectNext, nextCalled)
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestMediaTypeContentTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		expectNext  bool
		expectCode  int
	}{
		{"allowed vendor type", "application/vnd.api+json", `{"test": "data"}`, true, http.StatusOK},
		{"allowed with suffix", "application/custom+json", `{"test": "data"}`, true, http.StatusOK},
		{"disallowed plain json", "application/json", `{"test": "data"}`, false, http.StatusUnsupportedMediaType},
		{"disallowed text", "text/plain", "plain text", false, http.StatusUnsupportedMediaType},
		{"empty body skipped", "text/plain", "", true, http.StatusOK},
		{"case insensitive", "APPLICATION/VND.API+JSON", `{"test": "data"}`, true, http.StatusOK},
		{"charset parameter", "application/vnd.api+json; charset=utf-8", `{"test": "data"}`, true, http.StatusOK},
		{"boundary parameter", "multipart/form-data; boundary=----WebKit", "form data", false, http.StatusUnsupportedMediaType},
		{"whitespace in content-type", " application/vnd.api+json ", `{"test": "data"}`, true, http.StatusOK},
	}

	middleware := New(Config{
		AllowedTypes:        []string{"application/vnd.api+json", "application/*+json"},
		ValidateContentType: true,
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/test", nil)
			}
			if tt.contentType != "" {
				req.Header.Set(httpx.HeaderContentType, tt.contentType)
			}

			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			zhtest.AssertEqual(t, tt.expectNext, nextCalled)
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestMediaTypeWildcardPatterns(t *testing.T) {
	tests := []struct {
		name       string
		accept     string
		expectNext bool
	}{
		{"exact match", "application/vnd.api+json", true},
		{"company wildcard", "application/vnd.company.resource+json", true},
		{"other company", "application/vnd.other+json", true},
		{"no match wrong suffix", "application/vnd.api+xml", false},
		{"no match wrong type", "text/plain", false},
	}

	middleware := New(Config{AllowedTypes: []string{"application/vnd.*+json"}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(httpx.HeaderAccept, tt.accept)

			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			zhtest.AssertEqual(t, tt.expectNext, nextCalled)
		})
	}
}

func TestMediaTypeDefaultType(t *testing.T) {
	tests := []struct {
		name         string
		accept       string
		expectAccept string
	}{
		{"*/* gets default", "*/*", "application/vnd.api+json"},
		{"no accept gets default", "", "application/vnd.api+json"},
		{"specific accept preserved", "application/vnd.api.v2+json", "application/vnd.api.v2+json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := New(Config{
				AllowedTypes: []string{"application/vnd.api+json", "application/vnd.api.v2+json"},
				DefaultType:  "application/vnd.api+json",
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.accept != "" {
				req.Header.Set(httpx.HeaderAccept, tt.accept)
			}

			rr := httptest.NewRecorder()
			var receivedAccept string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedAccept = r.Header.Get(httpx.HeaderAccept)
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Status(http.StatusOK)
			zhtest.AssertEqual(t, tt.expectAccept, receivedAccept)
		})
	}
}

func TestMediaTypeNoValidation(t *testing.T) {
	middleware := New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderAccept, "text/plain")

	rr := httptest.NewRecorder()
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)

	zhtest.AssertTrue(t, nextCalled)
	zhtest.AssertWith(t, rr).Status(http.StatusOK)
}

func TestMediaTypeExcludedPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectNext bool
	}{
		{"excluded /health", "/health", true},
		{"excluded prefix /public/", "/public/css", true},
		{"not excluded /api", "/api", false},
	}

	middleware := New(Config{
		AllowedTypes:  []string{"application/json"},
		ExcludedPaths: []string{"/health", "/public/"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(httpx.HeaderAccept, "text/plain")

			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			zhtest.AssertEqual(t, tt.expectNext, nextCalled)
		})
	}
}

func TestMediaTypeBothExcludedAndIncludedPathsPanics(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			AllowedTypes:  []string{"application/json"},
			ExcludedPaths: []string{"/health"},
			IncludedPaths: []string{"/api"},
		})
	})
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		match   bool
	}{
		{"application/*", "application/json", true},
		{"application/*", "application/xml", true},
		{"application/*", "text/plain", false},
		{"application/vnd.*+json", "application/vnd.api+json", true},
		{"application/vnd.*+json", "application/vnd.company.resource+json", true},
		{"application/vnd.*+json", "application/vnd.api+xml", false},
		{"*+json", "application/json", false},
		{"*+json", "application/vnd.api+json", true},
		{"*+json", "application/xml", false},
		{"application/*+json", "application/json", false},
		{"application/*+json", "application/vnd.api+json", true},
		{"application/json*", "application/json", true},
		{"application/json*", "application/json-patch", true},
		{"application/*xml*", "application/xml", true},
		{"application/*xml*", "application/vnd.xml+json", true},
		{"*", "anything", true},
		{"*", "", true},
		{"", "", true},
		{"", "something", false},
		{"application/*/*", "application/api/v1", true},
		{"application/*/*", "application/api", false},
		{"*/*", "application/json", true},
		{"*/*", "text/plain", true},
		{"application/json", "application/json", true},
		{"application/json", "application/xml", false},
		{"application/**", "application/json", true},
		{"application/*", "application/", true},
		{"*/json", "application/json", true},
		{"*/json", "text/json", true},
		{"*/json", "application/xml", false},
		{"*/*+json", "application/vnd.api+json", true},
		{"*/*+json", "application/json", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			result := matchWildcard(tt.input, tt.pattern)
			zhtest.AssertEqual(t, tt.match, result)
		})
	}
}

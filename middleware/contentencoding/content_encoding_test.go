package contentencoding

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestContentEncodingValidation(t *testing.T) {
	tests := []struct {
		name            string
		contentEncoding string
		body            string
		expectNext      bool
		expectCode      int
	}{
		{"allowed gzip", "gzip", "test", true, http.StatusOK},
		{"allowed deflate", "deflate", "test", true, http.StatusOK},
		{"allowed uppercase", "GZIP", "test", true, http.StatusOK},
		{"allowed with spaces", " gzip ", "test", true, http.StatusOK},
		{"disallowed br", "br", "test", false, http.StatusUnsupportedMediaType},
		{"disallowed compress", "compress", "test", false, http.StatusUnsupportedMediaType},
		{"no encoding header", "", "test", true, http.StatusOK},
		{"empty body skipped", "br", "", true, http.StatusOK},
	}

	middleware := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/test", nil)
			}
			if tt.contentEncoding != "" {
				req.Header.Set(httpx.HeaderContentEncoding, tt.contentEncoding)
			}

			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestContentEncodingMultipleValues(t *testing.T) {
	tests := []struct {
		name       string
		encodings  []string
		expectNext bool
		expectCode int
	}{
		{"multiple allowed", []string{"gzip", "deflate"}, true, http.StatusOK},
		{"comma separated allowed", []string{"gzip, deflate"}, true, http.StatusOK},
		{"mixed allowed/disallowed", []string{"gzip", "br"}, false, http.StatusUnsupportedMediaType},
		{"comma separated mixed", []string{"gzip, br"}, false, http.StatusUnsupportedMediaType},
		{"all disallowed", []string{"br", "compress"}, false, http.StatusUnsupportedMediaType},
	}

	middleware := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
			for _, encoding := range tt.encodings {
				req.Header.Add(httpx.HeaderContentEncoding, encoding)
			}
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestContentEncodingCustomConfig(t *testing.T) {
	tests := []struct {
		name       string
		encoding   string
		expectNext bool
		expectCode int
	}{
		{"custom allowed br", "br", true, http.StatusOK},
		{"custom allowed gzip", "gzip", true, http.StatusOK},
		{"custom disallowed deflate", "deflate", false, http.StatusUnsupportedMediaType},
	}

	middleware := New(Config{Encodings: []string{"br", "gzip"}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
			req.Header.Set(httpx.HeaderContentEncoding, tt.encoding)
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestContentEncodingExcludedPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		encoding   string
		expectNext bool
		expectCode int
	}{
		{"excluded exact", "/health", "br", true, http.StatusOK},
		{"excluded prefix", "/api/webhooks/github", "br", true, http.StatusOK},
		{"not excluded", "/api/users", "br", false, http.StatusUnsupportedMediaType},
	}

	middleware := New(Config{
		Encodings:     []string{"gzip"},
		ExcludedPaths: []string{"/health", "/api/webhooks/"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("test"))
			req.Header.Set(httpx.HeaderContentEncoding, tt.encoding)
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestContentEncodingHTTPMethods(t *testing.T) {
	middleware := New()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", strings.NewReader("test"))
			req.Header.Set(httpx.HeaderContentEncoding, "gzip")
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if !nextCalled {
				t.Errorf("handler should be called for method %s", method)
			}
			zhtest.AssertWith(t, rr).Status(http.StatusOK)
		})
	}
}

func TestContentEncodingNilEncodingsFallback(t *testing.T) {
	middleware := New(Config{Encodings: nil}) // Explicitly set to nil
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
	req.Header.Set(httpx.HeaderContentEncoding, "gzip") // Should be allowed by default config
	rr := httptest.NewRecorder()
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("handler should be called with default encodings fallback")
	}
	zhtest.AssertWith(t, rr).Status(http.StatusOK)
}

func TestContentEncodingNilExcludedPathsFallback(t *testing.T) {
	middleware := New(Config{
		Encodings:     []string{"gzip"},
		ExcludedPaths: nil,
	})
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
	req.Header.Set(httpx.HeaderContentEncoding, "br")
	rr := httptest.NewRecorder()
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)

	if nextCalled {
		t.Error("handler should not be called with disallowed encoding")
	}
	zhtest.AssertWith(t, rr).Status(http.StatusUnsupportedMediaType)
}

func TestContentEncodingIncludedPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		encoding   string
		expectNext bool
		expectCode int
	}{
		{"allowed prefix", "/api/users", "br", false, http.StatusUnsupportedMediaType},
		{"allowed exact", "/upload", "br", false, http.StatusUnsupportedMediaType},
		{"not allowed", "/other", "br", true, http.StatusOK},
	}

	middleware := New(Config{
		Encodings:     []string{"gzip"},
		IncludedPaths: []string{"/api/", "/upload"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("test"))
			req.Header.Set(httpx.HeaderContentEncoding, tt.encoding)
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
		})
	}
}

func TestContentEncodingBothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = New(Config{
		Encodings:     []string{"gzip"},
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api"},
	})
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
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

	middleware := ContentEncoding()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(http.MethodPost, "/test", nil)
			}
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
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

	middleware := ContentEncoding()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
			for _, encoding := range tt.encodings {
				req.Header.Add("Content-Encoding", encoding)
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

	middleware := ContentEncoding(config.WithContentEncodingEncodings([]string{"br", "gzip"}))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
			req.Header.Set("Content-Encoding", tt.encoding)
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

func TestContentEncodingExemptPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		encoding   string
		expectNext bool
		expectCode int
	}{
		{"exempt exact", "/health", "br", true, http.StatusOK},
		{"exempt prefix", "/api/webhooks/github", "br", true, http.StatusOK},
		{"not exempt", "/api/users", "br", false, http.StatusUnsupportedMediaType},
	}

	middleware := ContentEncoding(
		config.WithContentEncodingEncodings([]string{"gzip"}),
		config.WithContentEncodingExemptPaths([]string{"/health", "/api/webhooks/"}),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("test"))
			req.Header.Set("Content-Encoding", tt.encoding)
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
	middleware := ContentEncoding()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", strings.NewReader("test"))
			req.Header.Set("Content-Encoding", "gzip")
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
	middleware := ContentEncoding(config.WithContentEncodingEncodings(nil)) // Explicitly set to nil
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
	req.Header.Set("Content-Encoding", "gzip") // Should be allowed by default config
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

func TestContentEncodingNilExemptPathsFallback(t *testing.T) {
	middleware := ContentEncoding(
		config.WithContentEncodingEncodings([]string{"gzip"}),
		config.WithContentEncodingExemptPaths(nil),
	)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
	req.Header.Set("Content-Encoding", "br")
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

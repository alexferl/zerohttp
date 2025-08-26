package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestContentCharset(t *testing.T) {
	tests := []struct {
		name        string
		config      func() func(http.Handler) http.Handler
		contentType string
		expectNext  bool
		expectCode  int
	}{
		{
			name:        "utf-8 charset allowed",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "application/json; charset=utf-8",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name:        "UTF-8 uppercase allowed",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "application/json; charset=UTF-8",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name:        "no charset allowed",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "application/json",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name:        "empty content-type allowed",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name:        "iso-8859-1 not allowed by default",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "text/html; charset=iso-8859-1",
			expectNext:  false,
			expectCode:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "charset with spaces",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "text/plain; charset = utf-8 ; boundary=something",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name:        "multiple parameters with charset",
			config:      func() func(http.Handler) http.Handler { return ContentCharset() },
			contentType: "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW; charset=utf-8",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name: "custom config allows iso-8859-1",
			config: func() func(http.Handler) http.Handler {
				return ContentCharset(config.WithContentCharsetCharsets([]string{"utf-8", "iso-8859-1", "windows-1252"}))
			},
			contentType: "text/html; charset=iso-8859-1",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name: "custom config case insensitive",
			config: func() func(http.Handler) http.Handler {
				return ContentCharset(config.WithContentCharsetCharsets([]string{"utf-8", "iso-8859-1"}))
			},
			contentType: "text/plain; charset=ISO-8859-1",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
		{
			name: "custom config rejects ascii",
			config: func() func(http.Handler) http.Handler {
				return ContentCharset(config.WithContentCharsetCharsets([]string{"utf-8", "iso-8859-1"}))
			},
			contentType: "text/plain; charset=ascii",
			expectNext:  false,
			expectCode:  http.StatusUnsupportedMediaType,
		},
		{
			name: "empty config rejects all",
			config: func() func(http.Handler) http.Handler {
				return ContentCharset(config.WithContentCharsetCharsets([]string{}))
			},
			contentType: "application/json; charset=utf-8",
			expectNext:  false,
			expectCode:  http.StatusUnsupportedMediaType,
		},
		{
			name: "nil config uses defaults",
			config: func() func(http.Handler) http.Handler {
				return ContentCharset(config.WithContentCharsetCharsets(nil))
			},
			contentType: "application/json; charset=utf-8",
			expectNext:  true,
			expectCode:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := tt.config()
			req := httptest.NewRequest("POST", "/test", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if nextCalled != tt.expectNext {
				t.Errorf("Expected nextCalled=%v, got nextCalled=%v", tt.expectNext, nextCalled)
			}
			if rr.Code != tt.expectCode {
				t.Errorf("Expected status code %d, got %d", tt.expectCode, rr.Code)
			}
		})
	}
}

func TestContentCharsetHTTPMethods(t *testing.T) {
	middleware := ContentCharset()
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			rr := httptest.NewRecorder()
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})
			middleware(next).ServeHTTP(rr, req)

			if !nextCalled {
				t.Errorf("Next handler should be called for method %s", method)
			}
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200 for method %s, got %d", method, rr.Code)
			}
		})
	}
}

func TestContentEncodingFunction(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		charsets    []string
		expected    bool
	}{
		{
			name:        "exact charset match",
			contentType: "text/plain; charset=utf-8",
			charsets:    []string{"utf-8"},
			expected:    true,
		},
		{
			name:        "case insensitive match",
			contentType: "text/plain; charset=UTF-8",
			charsets:    []string{"utf-8"},
			expected:    true,
		},
		{
			name:        "no charset in content-type",
			contentType: "application/json",
			charsets:    []string{"", "utf-8"},
			expected:    true,
		},
		{
			name:        "charset not in allowed list",
			contentType: "text/html; charset=iso-8859-1",
			charsets:    []string{"utf-8"},
			expected:    false,
		},
		{
			name:        "empty content type",
			contentType: "",
			charsets:    []string{""},
			expected:    true,
		},
		{
			name:        "charset with spaces",
			contentType: "text/plain; charset = utf-8 ",
			charsets:    []string{"utf-8"},
			expected:    true,
		},
		{
			name:        "multiple parameters",
			contentType: "multipart/form-data; boundary=abc123; charset=utf-8; foo=bar",
			charsets:    []string{"utf-8"},
			expected:    true,
		},
		{
			name:        "charset first parameter",
			contentType: "text/plain; charset=utf-8; boundary=abc123",
			charsets:    []string{"utf-8"},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contentEncoding(tt.contentType, tt.charsets...)
			if result != tt.expected {
				t.Errorf("contentEncoding(%q, %v) = %v, expected %v",
					tt.contentType, tt.charsets, result, tt.expected)
			}
		})
	}
}

func TestSplitFunction(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		sep     string
		expectA string
		expectB string
	}{
		{
			name:    "basic split",
			str:     "hello;world",
			sep:     ";",
			expectA: "hello",
			expectB: "world",
		},
		{
			name:    "split with spaces",
			str:     "hello ; world ",
			sep:     ";",
			expectA: "hello",
			expectB: "world",
		},
		{
			name:    "no separator found",
			str:     "hello world",
			sep:     ";",
			expectA: "hello world",
			expectB: "",
		},
		{
			name:    "separator at start",
			str:     ";world",
			sep:     ";",
			expectA: "",
			expectB: "world",
		},
		{
			name:    "separator at end",
			str:     "hello;",
			sep:     ";",
			expectA: "hello",
			expectB: "",
		},
		{
			name:    "empty string",
			str:     "",
			sep:     ";",
			expectA: "",
			expectB: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := split(tt.str, tt.sep)
			if a != tt.expectA {
				t.Errorf("Expected first part %q, got %q", tt.expectA, a)
			}
			if b != tt.expectB {
				t.Errorf("Expected second part %q, got %q", tt.expectB, b)
			}
		})
	}
}

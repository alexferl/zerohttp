package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestContentTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		expectNext  bool
		expectCode  int
	}{
		{"allowed application/json", "application/json", `{"test": "data"}`, true, http.StatusOK},
		{"allowed application/x-www-form-urlencoded", "application/x-www-form-urlencoded", "key=value", true, http.StatusOK},
		{"allowed multipart/form-data", "multipart/form-data", "form data", true, http.StatusOK},
		{"allowed with charset parameter", "application/json; charset=utf-8", `{"test": "data"}`, true, http.StatusOK},
		{"allowed with boundary parameter", "multipart/form-data; boundary=----WebKitFormBoundary", "form data", true, http.StatusOK},
		{"allowed case insensitive", "APPLICATION/JSON", `{"test": "data"}`, true, http.StatusOK},
		{"allowed with spaces", " application/json ", `{"test": "data"}`, true, http.StatusOK},
		{"disallowed text/plain", "text/plain", "plain text", false, http.StatusUnsupportedMediaType},
		{"disallowed application/xml", "application/xml", "data", false, http.StatusUnsupportedMediaType},
		{"no content-type header", "", "some data", false, http.StatusUnsupportedMediaType},
		{"empty body skipped", "text/plain", "", true, http.StatusOK},
		{"json with complex parameters", "application/json; charset=utf-8; boundary=something", "data", true, http.StatusOK},
		{"form with boundary and charset", "multipart/form-data; boundary=----WebKit; charset=utf-8", "data", true, http.StatusOK},
		{"invalid type with valid parameters", "text/plain; charset=utf-8", "data", false, http.StatusUnsupportedMediaType},
	}

	middleware := ContentType()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest("POST", "/test", nil)
			}
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
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			if rr.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, rr.Code)
			}
		})
	}
}

func TestContentTypeCustomConfig(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expectNext  bool
		expectCode  int
	}{
		{"custom allowed text/plain", "text/plain", true, http.StatusOK},
		{"custom allowed application/xml", "application/xml", true, http.StatusOK},
		{"custom disallowed application/json", "application/json", false, http.StatusUnsupportedMediaType},
	}

	middleware := ContentType(config.WithContentTypeContentTypes([]string{"text/plain", "application/xml"}))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader("test data"))
			req.Header.Set("Content-Type", tt.contentType)
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
			if rr.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, rr.Code)
			}
		})
	}
}

func TestContentTypeExemptPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectNext bool
		expectCode int
	}{
		{"exempt exact path /health", "/health", true, http.StatusOK},
		{"exempt exact path /api/upload", "/api/upload", true, http.StatusOK},
		{"exempt prefix /webhooks/", "/webhooks/", true, http.StatusOK},
		{"exempt prefix /webhooks/github", "/webhooks/github", true, http.StatusOK},
		{"not exempt /api/users", "/api/users", false, http.StatusUnsupportedMediaType},
	}

	middleware := ContentType(
		config.WithContentTypeContentTypes([]string{"application/json"}),
		config.WithContentTypeExemptPaths([]string{"/health", "/webhooks/", "/api/upload"}),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.path, strings.NewReader("test data"))
			req.Header.Set("Content-Type", "text/plain")
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
			if rr.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, rr.Code)
			}
		})
	}
}

func TestContentTypeHTTPMethods(t *testing.T) {
	middleware := ContentType()
	methods := []string{"POST", "PUT", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", strings.NewReader(`{"test": "data"}`))
			req.Header.Set("Content-Type", "application/json")
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
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200 for method %s, got %d", method, rr.Code)
			}
		})
	}
}

func TestContentTypeGETRequests(t *testing.T) {
	middleware := ContentType()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	middleware(next).ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("GET request should skip content type validation")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET request, got %d", rr.Code)
	}
}

func TestContentTypeConfigFallbacks(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		middleware := ContentType(config.WithContentTypeContentTypes([]string{}))
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})
		middleware(next).ServeHTTP(rr, req)

		if nextCalled {
			t.Error("next handler should not be called with empty content types config")
		}
		if rr.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected status 415, got %d", rr.Code)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		middleware := ContentType(config.WithContentTypeContentTypes(nil))
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"test": "data"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		middleware(next).ServeHTTP(rr, req)

		if !nextCalled {
			t.Error("next handler should be called with default config when nil")
		}
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("nil exempt paths fallback", func(t *testing.T) {
		middleware := ContentType(
			config.WithContentTypeContentTypes([]string{"application/json"}),
			config.WithContentTypeExemptPaths(nil),
		)
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
		req.Header.Set("Content-Type", "text/plain")
		rr := httptest.NewRecorder()
		called := false
		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})).ServeHTTP(rr, req)

		if called || rr.Code != http.StatusUnsupportedMediaType {
			t.Error("should fallback to default exempt paths and reject text/plain")
		}
	})
}

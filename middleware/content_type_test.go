package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
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

			if nextCalled != tt.expectNext {
				t.Errorf("expected nextCalled=%v, got %v", tt.expectNext, nextCalled)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)
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

	middleware := ContentType(config.ContentTypeConfig{ContentTypes: []string{"text/plain", "application/xml"}})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test data"))
			req.Header.Set(httpx.HeaderContentType, tt.contentType)
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

func TestContentTypeExcludedPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectNext bool
		expectCode int
	}{
		{"excluded exact path /health", "/health", true, http.StatusOK},
		{"excluded exact path /api/upload", "/api/upload", true, http.StatusOK},
		{"excluded prefix /webhooks/", "/webhooks/", true, http.StatusOK},
		{"excluded prefix /webhooks/github", "/webhooks/github", true, http.StatusOK},
		{"not excluded /api/users", "/api/users", false, http.StatusUnsupportedMediaType},
	}

	middleware := ContentType(config.ContentTypeConfig{
		ContentTypes:  []string{"application/json"},
		ExcludedPaths: []string{"/health", "/webhooks/", "/api/upload"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("test data"))
			req.Header.Set(httpx.HeaderContentType, httpx.MIMETextPlain)
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

func TestContentTypeHTTPMethods(t *testing.T) {
	middleware := ContentType()
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", strings.NewReader(`{"test": "data"}`))
			req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
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

func TestContentTypeGETRequests(t *testing.T) {
	middleware := ContentType()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderContentType, httpx.MIMETextPlain)
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
	zhtest.AssertWith(t, rr).Status(http.StatusOK)
}

func TestContentTypeConfigFallbacks(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		middleware := ContentType(config.ContentTypeConfig{ContentTypes: []string{}})
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		rr := httptest.NewRecorder()
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})
		middleware(next).ServeHTTP(rr, req)

		if nextCalled {
			t.Error("next handler should not be called with empty content types config")
		}
		zhtest.AssertWith(t, rr).Status(http.StatusUnsupportedMediaType)
	})

	t.Run("nil config", func(t *testing.T) {
		middleware := ContentType(config.ContentTypeConfig{ContentTypes: nil})
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"test": "data"}`))
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
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
		zhtest.AssertWith(t, rr).Status(http.StatusOK)
	})

	t.Run("nil excluded paths fallback", func(t *testing.T) {
		middleware := ContentType(config.ContentTypeConfig{
			ContentTypes:  []string{httpx.MIMEApplicationJSON},
			ExcludedPaths: nil,
		})
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test"))
		req.Header.Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		rr := httptest.NewRecorder()
		called := false
		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})).ServeHTTP(rr, req)

		if called || rr.Code != http.StatusUnsupportedMediaType {
			t.Error("should fallback to default excluded paths and reject text/plain")
		}
	})
}

func TestContentTypeIncludedPaths(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		expectNext bool
		expectCode int
	}{
		{"allowed exact /api", "/api", false, http.StatusUnsupportedMediaType},
		{"allowed prefix /api/", "/api/users", false, http.StatusUnsupportedMediaType},
		{"allowed exact /upload", "/upload", false, http.StatusUnsupportedMediaType},
		{"not allowed /other", "/other", true, http.StatusOK},
		{"not allowed /health", "/health", true, http.StatusOK},
	}

	middleware := ContentType(config.ContentTypeConfig{
		ContentTypes:  []string{httpx.MIMEApplicationJSON},
		IncludedPaths: []string{"/api/", "/api", "/upload"},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader("test data"))
			req.Header.Set(httpx.HeaderContentType, httpx.MIMETextPlain)
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

func TestContentTypeBothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = ContentType(config.ContentTypeConfig{
		ContentTypes:  []string{httpx.MIMEApplicationJSON},
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api"},
	})
}

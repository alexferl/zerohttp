package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestCORSSimpleRequest(t *testing.T) {
	tests := []struct {
		origin       string
		method       string
		expectOrigin string
		expectNext   bool
	}{
		{"https://example.com", http.MethodGet, "*", true},
		{"", http.MethodGet, "", true},
		{"https://api.example.com", http.MethodPost, "*", true},
	}
	mw := CORS()
	for _, tt := range tests {
		t.Run(tt.origin+"-"+tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()
			called := false
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			if called != tt.expectNext {
				t.Errorf("expected called=%v, got %v", tt.expectNext, called)
			}
			zhtest.AssertWith(t, rr).Header("Access-Control-Allow-Origin", tt.expectOrigin)
		})
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	tests := []struct {
		name, origin, method, headers string
		expectCode                    int
		expectNext                    bool
		checkProblemDetail            bool
		checkAllowHeader              bool
	}{
		{"valid", "https://example.com", http.MethodPost, "Content-Type", http.StatusNoContent, false, false, false},
		{"multiple headers", "https://example.com", http.MethodPut, "Content-Type, Authorization", http.StatusNoContent, false, false, false},
		{"bad method", "https://example.com", http.MethodTrace, "", http.StatusMethodNotAllowed, false, true, true},
		{"bad header", "https://example.com", http.MethodPost, "X-Custom-Header", http.StatusForbidden, false, true, false},
		{"no origin", "", http.MethodPost, "Content-Type", http.StatusNoContent, false, false, false},
	}
	mw := CORS()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.method != "" {
				req.Header.Set("Access-Control-Request-Method", tt.method)
			}
			if tt.headers != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.headers)
			}
			rr := httptest.NewRecorder()
			called := false
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			})).ServeHTTP(rr, req)

			if called != tt.expectNext {
				t.Errorf("expected called=%v, got %v", tt.expectNext, called)
			}
			zhtest.AssertWith(t, rr).Status(tt.expectCode)

			if tt.checkProblemDetail {
				zhtest.AssertWith(t, rr).IsProblemDetail()
			}
			if tt.checkAllowHeader {
				zhtest.AssertWith(t, rr).HeaderExists("Allow")
			}
		})
	}
}

func TestCORSCustomOrigins(t *testing.T) {
	tests := []struct {
		origin, expectOrigin string
	}{
		{"https://example.com", "https://example.com"},
		{"https://api.example.com", "https://api.example.com"},
		{"https://evil.com", ""},
		{"HTTPS://EXAMPLE.COM", "HTTPS://EXAMPLE.COM"},
	}
	mw := CORS(config.CORSConfig{AllowedOrigins: []string{"https://example.com", "https://api.example.com"}})
	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header("Access-Control-Allow-Origin", tt.expectOrigin)
		})
	}
}

func TestCORSCredentials(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Header("Access-Control-Allow-Origin", "https://example.com").
		Header("Access-Control-Allow-Credentials", "true")
}

func TestCORSCredentialsWithWildcard(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Header("Access-Control-Allow-Origin", "https://example.com").
		Header("Access-Control-Allow-Credentials", "true")
}

func TestCORSExposedHeaders(t *testing.T) {
	mw := CORS(config.CORSConfig{ExposedHeaders: []string{"X-Total-Count", "X-Page-Count"}})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Header("Access-Control-Expose-Headers", "X-Total-Count, X-Page-Count")
}

func TestCORSOptionsPassthrough(t *testing.T) {
	mw := CORS(config.CORSConfig{OptionsPassthrough: true})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler called with OptionsPassthrough=true")
	}
	zhtest.AssertWith(t, rr).Status(http.StatusOK)
}

func TestCORSExemptPaths(t *testing.T) {
	tests := []struct {
		path       string
		expectCORS bool
	}{
		{"/skip-cors", false},
		{"/no-cors", false},
		{"/api/users", true},
	}
	mw := CORS(config.CORSConfig{ExemptPaths: []string{"/skip-cors", "/no-cors"}})
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Origin", "https://example.com")
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			corsOrigin := rr.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS && corsOrigin == "" {
				t.Error("expected CORS headers")
			} else if !tt.expectCORS && corsOrigin != "" {
				t.Error("expected no CORS headers for exempt path")
			}
		})
	}
}

func TestCORSCustomConfig(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins: []string{"https://myapp.com"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://myapp.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Status(http.StatusNoContent).
		Header("Access-Control-Allow-Origin", "https://myapp.com").
		Header("Access-Control-Allow-Methods", "GET, POST").
		Header("Access-Control-Max-Age", "3600")
}

func TestCORSNilConfig(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins: nil,
		AllowedMethods: nil,
		AllowedHeaders: nil,
	})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Status(http.StatusNoContent).
		Header("Access-Control-Allow-Origin", "*")

	if methods := rr.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, http.MethodGet) {
		t.Errorf("expected methods to contain 'GET', got '%s'", methods)
	}
}

func TestCORSNilExemptPathsFallback(t *testing.T) {
	mw := CORS(config.CORSConfig{ExemptPaths: nil})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called {
		t.Error("should fallback to default exempt paths and process CORS")
	}
	zhtest.AssertWith(t, rr).Header("Access-Control-Allow-Origin", "*")
}

func TestCORSZeroMaxAgeFallback(t *testing.T) {
	mw := CORS(config.CORSConfig{MaxAge: 0})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Header("Access-Control-Max-Age", "86400")
}

func TestCORSNoOriginOptionsPassthrough(t *testing.T) {
	mw := CORS(config.CORSConfig{OptionsPassthrough: true})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called {
		t.Error("should pass OPTIONS without Origin with passthrough")
	}
	zhtest.AssertWith(t, rr).Status(http.StatusOK)
}

func TestCORSDisallowedOriginOptionsPassthrough(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins:     []string{"https://allowed.com"},
		OptionsPassthrough: true,
	})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://notallowed.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called {
		t.Error("should pass OPTIONS with disallowed Origin and passthrough")
	}
	zhtest.AssertWith(t, rr).Status(http.StatusOK)
}

func TestCORSDisallowedOriginNoPassthrough(t *testing.T) {
	mw := CORS(config.CORSConfig{AllowedOrigins: []string{"https://allowed.com"}})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://notallowed.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	if called {
		t.Error("handler should not be called when origin disallowed and passthrough is false")
	}
	zhtest.AssertWith(t, rr).Status(http.StatusNoContent)
}

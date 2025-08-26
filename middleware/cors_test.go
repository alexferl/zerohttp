package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestCORSSimpleRequest(t *testing.T) {
	tests := []struct {
		origin       string
		method       string
		expectOrigin string
		expectNext   bool
	}{
		{"https://example.com", "GET", "*", true},
		{"", "GET", "", true},
		{"https://api.example.com", "POST", "*", true},
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
			if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != tt.expectOrigin {
				t.Errorf("expected origin '%s', got '%s'", tt.expectOrigin, origin)
			}
		})
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	tests := []struct {
		name, origin, method, headers string
		expectCode                    int
		expectNext                    bool
	}{
		{"valid", "https://example.com", "POST", "Content-Type", http.StatusNoContent, false},
		{"multiple headers", "https://example.com", "PUT", "Content-Type, Authorization", http.StatusNoContent, false},
		{"bad method", "https://example.com", "TRACE", "", http.StatusMethodNotAllowed, false},
		{"bad header", "https://example.com", "POST", "X-Custom-Header", http.StatusForbidden, false},
		{"no origin", "", "POST", "Content-Type", http.StatusNoContent, false},
	}
	mw := CORS()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("OPTIONS", "/test", nil)
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

			if called != tt.expectNext || rr.Code != tt.expectCode {
				t.Errorf("expected called=%v code=%d, got called=%v code=%d",
					tt.expectNext, tt.expectCode, called, rr.Code)
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
	mw := CORS(config.WithCORSAllowedOrigins([]string{"https://example.com", "https://api.example.com"}))
	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != tt.expectOrigin {
				t.Errorf("expected origin '%s', got '%s'", tt.expectOrigin, origin)
			}
		})
	}
}

func TestCORSCredentials(t *testing.T) {
	mw := CORS(
		config.WithCORSAllowedOrigins([]string{"https://example.com"}),
		config.WithCORSAllowCredentials(true),
	)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
		t.Errorf("expected specific origin, got '%s'", origin)
	}
	if creds := rr.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected credentials 'true', got '%s'", creds)
	}
}

func TestCORSCredentialsWithWildcard(t *testing.T) {
	mw := CORS(config.WithCORSAllowedOrigins([]string{"*"}), config.WithCORSAllowCredentials(true))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
		t.Errorf("expected specific origin with wildcard + credentials, got '%s'", origin)
	}
	if creds := rr.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected credentials 'true', got '%s'", creds)
	}
}

func TestCORSExposedHeaders(t *testing.T) {
	mw := CORS(config.WithCORSExposedHeaders([]string{"X-Total-Count", "X-Page-Count"}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	expected := "X-Total-Count, X-Page-Count"
	if exposed := rr.Header().Get("Access-Control-Expose-Headers"); exposed != expected {
		t.Errorf("expected '%s', got '%s'", expected, exposed)
	}
}

func TestCORSOptionsPassthrough(t *testing.T) {
	mw := CORS(config.WithCORSOptionsPassthrough(true))
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called || rr.Code != http.StatusOK {
		t.Error("expected handler called with OptionsPassthrough=true")
	}
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
	mw := CORS(config.WithCORSExemptPaths([]string{"/skip-cors", "/no-cors"}))
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
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
	mw := CORS(
		config.WithCORSAllowedOrigins([]string{"https://myapp.com"}),
		config.WithCORSAllowedMethods([]string{"GET", "POST"}),
		config.WithCORSAllowedHeaders([]string{"Content-Type"}),
		config.WithCORSMaxAge(3600),
	)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://myapp.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "https://myapp.com" {
		t.Errorf("expected myapp.com, got '%s'", origin)
	}
	if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST" {
		t.Errorf("expected 'GET, POST', got '%s'", methods)
	}
	if maxAge := rr.Header().Get("Access-Control-Max-Age"); maxAge != "3600" {
		t.Errorf("expected '3600', got '%s'", maxAge)
	}
}

func TestCORSNilConfig(t *testing.T) {
	mw := CORS(
		config.WithCORSAllowedOrigins(nil),
		config.WithCORSAllowedMethods(nil),
		config.WithCORSAllowedHeaders(nil),
	)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected '*' from defaults, got '%s'", origin)
	}
	if methods := rr.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "GET") {
		t.Errorf("expected methods to contain 'GET', got '%s'", methods)
	}
}

func TestCORSNilExemptPathsFallback(t *testing.T) {
	mw := CORS(config.WithCORSExemptPaths(nil))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called || rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("should fallback to default exempt paths and process CORS")
	}
}

func TestCORSZeroMaxAgeFallback(t *testing.T) {
	mw := CORS(config.WithCORSMaxAge(0))
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Error("should fallback to default max age 86400")
	}
}

func TestCORSNoOriginOptionsPassthrough(t *testing.T) {
	mw := CORS(config.WithCORSOptionsPassthrough(true))
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called || rr.Code != http.StatusOK {
		t.Error("should pass OPTIONS without Origin with passthrough")
	}
}

func TestCORSDisallowedOriginOptionsPassthrough(t *testing.T) {
	mw := CORS(
		config.WithCORSAllowedOrigins([]string{"https://allowed.com"}),
		config.WithCORSOptionsPassthrough(true),
	)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://notallowed.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called || rr.Code != http.StatusOK {
		t.Error("should pass OPTIONS with disallowed Origin and passthrough")
	}
}

func TestCORSDisallowedOriginNoPassthrough(t *testing.T) {
	mw := CORS(config.WithCORSAllowedOrigins([]string{"https://allowed.com"}))
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://notallowed.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	if called {
		t.Error("handler should not be called when origin disallowed and passthrough is false")
	}
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}
}

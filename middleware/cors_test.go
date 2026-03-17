package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
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
				req.Header.Set(httpx.HeaderOrigin, tt.origin)
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
			zhtest.AssertWith(t, rr).Header(httpx.HeaderAccessControlAllowOrigin, tt.expectOrigin)
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
				req.Header.Set(httpx.HeaderOrigin, tt.origin)
			}
			if tt.method != "" {
				req.Header.Set(httpx.HeaderAccessControlRequestMethod, tt.method)
			}
			if tt.headers != "" {
				req.Header.Set(httpx.HeaderAccessControlRequestHeaders, tt.headers)
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
				// Test JSON response with Accept header
				req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
				rr = httptest.NewRecorder()
				mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
				zhtest.AssertWith(t, rr).IsProblemDetail()

				// Test plain text response without Accept header
				req = httptest.NewRequest(http.MethodOptions, "/test", nil)
				if tt.origin != "" {
					req.Header.Set(httpx.HeaderOrigin, tt.origin)
				}
				if tt.method != "" {
					req.Header.Set(httpx.HeaderAccessControlRequestMethod, tt.method)
				}
				if tt.headers != "" {
					req.Header.Set(httpx.HeaderAccessControlRequestHeaders, tt.headers)
				}
				rr = httptest.NewRecorder()
				mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
				zhtest.AssertWith(t, rr).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
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
			req.Header.Set(httpx.HeaderOrigin, tt.origin)
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			zhtest.AssertWith(t, rr).Header(httpx.HeaderAccessControlAllowOrigin, tt.expectOrigin)
		})
	}
}

func TestCORSCredentials(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Header(httpx.HeaderAccessControlAllowOrigin, "https://example.com").
		Header(httpx.HeaderAccessControlAllowCredentials, "true")
}

func TestCORSCredentialsWithWildcard(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Header(httpx.HeaderAccessControlAllowOrigin, "https://example.com").
		Header(httpx.HeaderAccessControlAllowCredentials, "true")
}

func TestCORSExposedHeaders(t *testing.T) {
	mw := CORS(config.CORSConfig{ExposedHeaders: []string{"X-Total-Count", "X-Page-Count"}})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Header(httpx.HeaderAccessControlExposeHeaders, "X-Total-Count, X-Page-Count")
}

func TestCORSOptionsPassthrough(t *testing.T) {
	mw := CORS(config.CORSConfig{OptionsPassthrough: true})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	req.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodPost)
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
			req.Header.Set(httpx.HeaderOrigin, "https://example.com")
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			corsOrigin := rr.Header().Get(httpx.HeaderAccessControlAllowOrigin)
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
	req.Header.Set(httpx.HeaderOrigin, "https://myapp.com")
	req.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodPost)
	req.Header.Set(httpx.HeaderAccessControlRequestHeaders, "Content-Type")
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Status(http.StatusNoContent).
		Header(httpx.HeaderAccessControlAllowOrigin, "https://myapp.com").
		Header(httpx.HeaderAccessControlAllowMethods, "GET, POST").
		Header(httpx.HeaderAccessControlMaxAge, "3600")
}

func TestCORSNilConfig(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowedOrigins: nil,
		AllowedMethods: nil,
		AllowedHeaders: nil,
	})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	req.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodGet)
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).
		Status(http.StatusNoContent).
		Header(httpx.HeaderAccessControlAllowOrigin, "*")

	if methods := rr.Header().Get(httpx.HeaderAccessControlAllowMethods); !strings.Contains(methods, http.MethodGet) {
		t.Errorf("expected methods to contain 'GET', got '%s'", methods)
	}
}

func TestCORSNilExemptPathsFallback(t *testing.T) {
	mw := CORS(config.CORSConfig{ExemptPaths: nil})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	rr := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if !called {
		t.Error("should fallback to default exempt paths and process CORS")
	}
	zhtest.AssertWith(t, rr).Header(httpx.HeaderAccessControlAllowOrigin, "*")
}

func TestCORSZeroMaxAgeFallback(t *testing.T) {
	mw := CORS(config.CORSConfig{MaxAge: 0})
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://example.com")
	req.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodGet)
	rr := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)

	zhtest.AssertWith(t, rr).Header(httpx.HeaderAccessControlMaxAge, "86400")
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
	req.Header.Set(httpx.HeaderOrigin, "https://notallowed.com")
	req.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodPost)
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
	req.Header.Set(httpx.HeaderOrigin, "https://notallowed.com")
	req.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodPost)
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

func TestCORS_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := CORS(config.CORSConfig{
		AllowedOrigins: []string{"https://allowed.com"},
	})

	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	wrapped := metricsMw(handler)

	// Test preflight request
	req1 := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req1.Header.Set("Origin", "https://allowed.com")
	req1.Header.Set(httpx.HeaderAccessControlRequestMethod, http.MethodPost)
	rr1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr1.Code)
	}

	// Test allowed origin
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Origin", "https://allowed.com")
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}

	// Test rejected origin
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.Header.Set("Origin", "https://rejected.com")
	rr3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr3.Code)
	}

	// Check metrics
	families := reg.Gather()

	var preflightCounter *metrics.MetricFamily
	var originCounter *metrics.MetricFamily
	for _, f := range families {
		switch f.Name {
		case "cors_preflight_requests_total":
			preflightCounter = &f
		case "cors_requests_total":
			originCounter = &f
		}
	}

	if preflightCounter == nil {
		t.Fatal("expected cors_preflight_requests_total metric")
	}
	if originCounter == nil {
		t.Fatal("expected cors_requests_total metric")
	}

	// Should have 1 preflight
	preflightTotal := 0
	for _, m := range preflightCounter.Metrics {
		preflightTotal = int(m.Counter)
	}
	if preflightTotal != 1 {
		t.Errorf("expected 1 preflight, got %d", preflightTotal)
	}

	// Should have 1 allowed and 1 rejected
	allowed, rejected := 0, 0
	for _, m := range originCounter.Metrics {
		switch m.Labels["origin"] {
		case "allowed":
			allowed = int(m.Counter)
		case "rejected":
			rejected = int(m.Counter)
		}
	}
	if allowed != 1 {
		t.Errorf("expected 1 allowed, got %d", allowed)
	}
	if rejected != 1 {
		t.Errorf("expected 1 rejected, got %d", rejected)
	}
}

func TestCORSAllowOriginFunc(t *testing.T) {
	tests := []struct {
		name          string
		origin        string
		validator     config.OriginValidator
		expectAllowed bool
		expectVary    bool
	}{
		{
			name:   "allowed by function",
			origin: "https://app.example.com",
			validator: func(origin string) bool {
				return strings.HasSuffix(origin, ".example.com")
			},
			expectAllowed: true,
			expectVary:    true,
		},
		{
			name:   "rejected by function",
			origin: "https://evil.com",
			validator: func(origin string) bool {
				return strings.HasSuffix(origin, ".example.com")
			},
			expectAllowed: false,
			expectVary:    true,
		},
		{
			name:   "function takes precedence over allowed origins",
			origin: "https://other.com",
			validator: func(origin string) bool {
				return origin == "https://other.com"
			},
			expectAllowed: true,
			expectVary:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := CORS(config.CORSConfig{
				AllowedOrigins:  []string{"https://example.com"}, // Should be ignored when AllowOriginFunc is set
				AllowOriginFunc: tt.validator,
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set(httpx.HeaderOrigin, tt.origin)
			rr := httptest.NewRecorder()

			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			varyHeader := rr.Header().Get("Vary")
			if tt.expectVary && varyHeader != "Origin" {
				t.Errorf("expected Vary: Origin, got %s", varyHeader)
			}
			if !tt.expectVary && varyHeader == "Origin" {
				t.Errorf("unexpected Vary: Origin header")
			}

			allowOrigin := rr.Header().Get(httpx.HeaderAccessControlAllowOrigin)
			if tt.expectAllowed && allowOrigin == "" {
				t.Errorf("expected Access-Control-Allow-Origin header, got none")
			}
			if !tt.expectAllowed && allowOrigin != "" {
				t.Errorf("expected no Access-Control-Allow-Origin header, got %s", allowOrigin)
			}
		})
	}
}

func TestCORSCustomOriginFuncWithCredentials(t *testing.T) {
	mw := CORS(config.CORSConfig{
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "https://")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httpx.HeaderOrigin, "https://app.example.com")
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	// When credentials are allowed and using custom validator, should echo origin
	if got := rr.Header().Get(httpx.HeaderAccessControlAllowOrigin); got != "https://app.example.com" {
		t.Errorf("expected origin echo with credentials, got %s", got)
	}

	if got := rr.Header().Get(httpx.HeaderAccessControlAllowCredentials); got != "true" {
		t.Errorf("expected credentials header, got %s", got)
	}

	if got := rr.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected Vary: Origin, got %s", got)
	}
}

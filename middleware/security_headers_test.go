package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func mustGetHeader(h http.Header, name string) string {
	return h.Get(name)
}

func TestSecurityHeaders_CustomConfig(t *testing.T) {
	type headerTest struct {
		name       string
		middleware func(http.Handler) http.Handler
		header     string
		expected   string
		prepReq    func(*http.Request)
	}
	tests := []headerTest{
		{
			name:       "Custom CSP",
			middleware: SecurityHeaders(config.WithSecurityHeadersCSP("default-src 'self'")),
			header:     "Content-Security-Policy",
			expected:   "default-src 'self'",
			prepReq:    func(r *http.Request) {},
		},
		{
			name: "CSP Report Only",
			middleware: SecurityHeaders(
				config.WithSecurityHeadersCSP("default-src 'self'"),
				config.WithSecurityHeadersCSPReportOnly(true),
			),
			header:   "Content-Security-Policy-Report-Only",
			expected: "default-src 'self'",
			prepReq:  func(r *http.Request) {},
		},
		{
			name:       "Custom Permissions Policy",
			middleware: SecurityHeaders(config.WithSecurityHeadersPermissionsPolicy("camera=(), microphone=()")),
			header:     "Permissions-Policy",
			expected:   "camera=(), microphone=()",
			prepReq:    func(r *http.Request) {},
		},
		{
			name:       "Custom Referrer Policy",
			middleware: SecurityHeaders(config.WithSecurityHeadersReferrerPolicy("strict-origin-when-cross-origin")),
			header:     "Referrer-Policy",
			expected:   "strict-origin-when-cross-origin",
			prepReq:    func(r *http.Request) {},
		},
		{
			name:       "Custom Server Header",
			middleware: SecurityHeaders(config.WithSecurityHeadersServer("MyCustomServer")),
			header:     "Server",
			expected:   "MyCustomServer",
			prepReq:    func(r *http.Request) {},
		},
		{
			name: "Cross-Origin policies",
			middleware: SecurityHeaders(
				config.WithSecurityHeadersCrossOriginEmbedderPolicy("unsafe-none"),
				config.WithSecurityHeadersCrossOriginOpenerPolicy("unsafe-none"),
				config.WithSecurityHeadersCrossOriginResourcePolicy("cross-origin"),
			),
			header:   "",
			expected: "",
			prepReq:  func(r *http.Request) {},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			tt.prepReq(req)
			tt.middleware(handler).ServeHTTP(rec, req)

			if tt.header != "" {
				got := mustGetHeader(rec.Header(), tt.header)
				if got != tt.expected {
					t.Errorf("%s: got %q, want %q", tt.header, got, tt.expected)
				}
			} else {
				expectedHeaders := map[string]string{
					"Cross-Origin-Embedder-Policy": "unsafe-none",
					"Cross-Origin-Opener-Policy":   "unsafe-none",
					"Cross-Origin-Resource-Policy": "cross-origin",
				}
				for header, expected := range expectedHeaders {
					got := mustGetHeader(rec.Header(), header)
					if got != expected {
						t.Errorf("%s: got %q, want %q", header, got, expected)
					}
				}
			}
		})
	}
}

func TestSecurityHeaders_HSTSWithNestedOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SecurityHeaders(
		config.WithSecurityHeadersHSTS(
			config.WithHSTSMaxAge(31536000),
			config.WithHSTSPreload(true),
		),
	)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	got := mustGetHeader(rec.Header(), "Strict-Transport-Security")
	want := "max-age=31536000; includeSubDomains; preload"
	if got != want {
		t.Errorf("Strict-Transport-Security: got %q, want %q", got, want)
	}
}

func TestSecurityHeaders_ExemptPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SecurityHeaders(
		config.WithSecurityHeadersCSP("default-src 'self'"),
		config.WithSecurityHeadersExemptPaths([]string{"/skipme"}),
	)(handler)

	req := httptest.NewRequest(http.MethodGet, "/skipme", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Security-Policy") != "" {
		t.Errorf("CSP should not be set for exempt path '/skipme'")
	}
}

func TestSecurityHeaders_DefaultValuesFill(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := SecurityHeaders()(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	keys := []string{
		"Content-Security-Policy", "Cross-Origin-Embedder-Policy", "Cross-Origin-Opener-Policy",
		"Cross-Origin-Resource-Policy", "Permissions-Policy", "Referrer-Policy",
		"X-Content-Type-Options", "X-Frame-Options",
	}
	for _, key := range keys {
		if rec.Header().Get(key) == "" {
			t.Errorf("Header %s should have default value", key)
		}
	}
}

func TestSecurityHeaders_EmptyServerHidesHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := SecurityHeaders(config.WithSecurityHeadersServer(""))(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Header().Get("Server") != "" {
		t.Errorf("Server should not be set when config.Server is empty, got %q", rec.Header().Get("Server"))
	}
}

func TestSecurityHeaders_ContentSecurityPolicyNotSet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	middleware := SecurityHeaders(config.WithSecurityHeadersCSP(""))(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	got := mustGetHeader(rec.Header(), "Content-Security-Policy")
	want := config.DefaultSecurityHeadersConfig.ContentSecurityPolicy
	if got != want {
		t.Errorf("If config is empty, should use default CSP: got %q, want %q", got, want)
	}
}

func TestSecurityHeaders_DefaultValueFill_All(t *testing.T) {
	tests := []struct {
		name     string
		option   func(fn http.Handler) http.Handler
		header   string
		expected string
	}{
		{
			"DefaultValueFill_ContentSecurityPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersCSP(""))(h)
			},
			"Content-Security-Policy",
			config.DefaultSecurityHeadersConfig.ContentSecurityPolicy,
		},
		{
			"DefaultValueFill_CrossOriginEmbedderPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersCrossOriginEmbedderPolicy(""))(h)
			},
			"Cross-Origin-Embedder-Policy",
			config.DefaultSecurityHeadersConfig.CrossOriginEmbedderPolicy,
		},
		{
			"DefaultValueFill_CrossOriginOpenerPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersCrossOriginOpenerPolicy(""))(h)
			},
			"Cross-Origin-Opener-Policy",
			config.DefaultSecurityHeadersConfig.CrossOriginOpenerPolicy,
		},
		{
			"DefaultValueFill_CrossOriginResourcePolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersCrossOriginResourcePolicy(""))(h)
			},
			"Cross-Origin-Resource-Policy",
			config.DefaultSecurityHeadersConfig.CrossOriginResourcePolicy,
		},
		{
			"DefaultValueFill_PermissionsPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersPermissionsPolicy(""))(h)
			},
			"Permissions-Policy",
			config.DefaultSecurityHeadersConfig.PermissionsPolicy,
		},
		{
			"DefaultValueFill_ReferrerPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersReferrerPolicy(""))(h)
			},
			"Referrer-Policy",
			config.DefaultSecurityHeadersConfig.ReferrerPolicy,
		},
		{
			"DefaultValueFill_XContentTypeOptions",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersXContentTypeOptions(""))(h)
			},
			"X-Content-Type-Options",
			config.DefaultSecurityHeadersConfig.XContentTypeOptions,
		},
		{
			"DefaultValueFill_XFrameOptions",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.WithSecurityHeadersXFrameOptions(""))(h)
			},
			"X-Frame-Options",
			config.DefaultSecurityHeadersConfig.XFrameOptions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			mw := tt.option(handler)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)

			want := tt.expected
			got := rec.Header().Get(tt.header)
			if got != want {
				t.Errorf("%s: got %q, want %q", tt.header, got, want)
			}
		})
	}
}

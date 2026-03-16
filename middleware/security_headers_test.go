package middleware

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

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
			middleware: SecurityHeaders(config.SecurityHeadersConfig{ContentSecurityPolicy: "default-src 'self'"}),
			header:     "Content-Security-Policy",
			expected:   "default-src 'self'",
			prepReq:    func(r *http.Request) {},
		},
		{
			name: "CSP Report Only",
			middleware: SecurityHeaders(config.SecurityHeadersConfig{
				ContentSecurityPolicy:           "default-src 'self'",
				ContentSecurityPolicyReportOnly: true,
			}),
			header:   "Content-Security-Policy-Report-Only",
			expected: "default-src 'self'",
			prepReq:  func(r *http.Request) {},
		},
		{
			name:       "Custom Permissions Policy",
			middleware: SecurityHeaders(config.SecurityHeadersConfig{PermissionsPolicy: "camera=(), microphone=()"}),
			header:     "Permissions-Policy",
			expected:   "camera=(), microphone=()",
			prepReq:    func(r *http.Request) {},
		},
		{
			name:       "Custom Referrer Policy",
			middleware: SecurityHeaders(config.SecurityHeadersConfig{ReferrerPolicy: "strict-origin-when-cross-origin"}),
			header:     "Referrer-Policy",
			expected:   "strict-origin-when-cross-origin",
			prepReq:    func(r *http.Request) {},
		},
		{
			name:       "Custom Server Header",
			middleware: SecurityHeaders(config.SecurityHeadersConfig{Server: "MyCustomServer"}),
			header:     "Server",
			expected:   "MyCustomServer",
			prepReq:    func(r *http.Request) {},
		},
		{
			name: "Cross-Origin policies",
			middleware: SecurityHeaders(config.SecurityHeadersConfig{
				CrossOriginEmbedderPolicy: "unsafe-none",
				CrossOriginOpenerPolicy:   "unsafe-none",
				CrossOriginResourcePolicy: "cross-origin",
			}),
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
			req := zhtest.NewRequest(http.MethodGet, "/").Build()
			tt.prepReq(req)
			w := zhtest.TestMiddlewareWithHandler(tt.middleware, handler, req)

			zhtest.AssertWith(t, w).Status(http.StatusOK)
			if tt.header != "" {
				zhtest.AssertWith(t, w).Header(tt.header, tt.expected)
			} else {
				zhtest.AssertWith(t, w).
					Header("Cross-Origin-Embedder-Policy", "unsafe-none").
					Header("Cross-Origin-Opener-Policy", "unsafe-none").
					Header("Cross-Origin-Resource-Policy", "cross-origin")
			}
		})
	}
}

func TestSecurityHeaders_HSTSWithNestedOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	req.TLS = &tls.ConnectionState{}
	w := zhtest.TestMiddlewareWithHandler(
		SecurityHeaders(config.SecurityHeadersConfig{
			StrictTransportSecurity: config.StrictTransportSecurity{
				MaxAge:         31536000,
				PreloadEnabled: true,
			},
		}),
		handler,
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
}

func TestSecurityHeaders_ExemptPaths(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/skipme").Build()
	w := zhtest.TestMiddlewareWithHandler(
		SecurityHeaders(config.SecurityHeadersConfig{
			ContentSecurityPolicy: "default-src 'self'",
			ExemptPaths:           []string{"/skipme"},
		}),
		handler,
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderNotExists(httpx.HeaderContentSecurityPolicy)
}

func TestSecurityHeaders_DefaultValuesFill(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(SecurityHeaders(), req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).
		HeaderExists(httpx.HeaderContentSecurityPolicy).
		HeaderExists("Cross-Origin-Embedder-Policy").
		HeaderExists("Cross-Origin-Opener-Policy").
		HeaderExists("Cross-Origin-Resource-Policy").
		HeaderExists("Permissions-Policy").
		HeaderExists("Referrer-Policy").
		HeaderExists("X-Content-Type-Options").
		HeaderExists("X-Frame-Options")
}

func TestSecurityHeaders_EmptyServerHidesHeader(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SecurityHeaders(config.SecurityHeadersConfig{Server: ""}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderNotExists("Server")
}

func TestSecurityHeaders_ContentSecurityPolicyNotSet(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SecurityHeaders(config.SecurityHeadersConfig{ContentSecurityPolicy: ""}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).Header(httpx.HeaderContentSecurityPolicy, config.DefaultSecurityHeadersConfig.ContentSecurityPolicy)
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
				return SecurityHeaders(config.SecurityHeadersConfig{ContentSecurityPolicy: ""})(h)
			},
			"Content-Security-Policy",
			config.DefaultSecurityHeadersConfig.ContentSecurityPolicy,
		},
		{
			"DefaultValueFill_CrossOriginEmbedderPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{CrossOriginEmbedderPolicy: ""})(h)
			},
			"Cross-Origin-Embedder-Policy",
			config.DefaultSecurityHeadersConfig.CrossOriginEmbedderPolicy,
		},
		{
			"DefaultValueFill_CrossOriginOpenerPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{CrossOriginOpenerPolicy: ""})(h)
			},
			"Cross-Origin-Opener-Policy",
			config.DefaultSecurityHeadersConfig.CrossOriginOpenerPolicy,
		},
		{
			"DefaultValueFill_CrossOriginResourcePolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{CrossOriginResourcePolicy: ""})(h)
			},
			"Cross-Origin-Resource-Policy",
			config.DefaultSecurityHeadersConfig.CrossOriginResourcePolicy,
		},
		{
			"DefaultValueFill_PermissionsPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{PermissionsPolicy: ""})(h)
			},
			"Permissions-Policy",
			config.DefaultSecurityHeadersConfig.PermissionsPolicy,
		},
		{
			"DefaultValueFill_ReferrerPolicy",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{ReferrerPolicy: ""})(h)
			},
			"Referrer-Policy",
			config.DefaultSecurityHeadersConfig.ReferrerPolicy,
		},
		{
			"DefaultValueFill_XContentTypeOptions",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{XContentTypeOptions: ""})(h)
			},
			"X-Content-Type-Options",
			config.DefaultSecurityHeadersConfig.XContentTypeOptions,
		},
		{
			"DefaultValueFill_XFrameOptions",
			func(h http.Handler) http.Handler {
				return SecurityHeaders(config.SecurityHeadersConfig{XFrameOptions: ""})(h)
			},
			"X-Frame-Options",
			config.DefaultSecurityHeadersConfig.XFrameOptions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
			req := zhtest.NewRequest(http.MethodGet, "/").Build()
			w := zhtest.TestMiddlewareWithHandler(tt.option, handler, req)

			zhtest.AssertWith(t, w).Status(http.StatusOK)
			zhtest.AssertWith(t, w).Header(tt.header, tt.expected)
		})
	}
}

func TestSecurityHeaders_CSPNonce(t *testing.T) {
	tests := []struct {
		name            string
		csp             string
		nonceEnabled    bool
		wantNonce       bool
		wantCSPContains string
	}{
		{
			name:            "Nonce generated and replaced",
			csp:             "script-src 'nonce-{{nonce}}'",
			nonceEnabled:    true,
			wantNonce:       true,
			wantCSPContains: "script-src 'nonce-",
		},
		{
			name:            "No nonce when disabled",
			csp:             "script-src 'nonce-{{nonce}}'",
			nonceEnabled:    false,
			wantNonce:       false,
			wantCSPContains: "script-src 'nonce-{{nonce}}'",
		},
		{
			name:            "No placeholder no nonce",
			csp:             "default-src 'self'",
			nonceEnabled:    true,
			wantNonce:       false,
			wantCSPContains: "default-src 'self'",
		},
		{
			name:            "Multiple placeholders replaced",
			csp:             "script-src 'nonce-{{nonce}}'; style-src 'nonce-{{nonce}}'",
			nonceEnabled:    true,
			wantNonce:       true,
			wantCSPContains: "'nonce-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedNonce string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedNonce = GetCSPNonce(r)
				w.WriteHeader(http.StatusOK)
			})

			req := zhtest.NewRequest(http.MethodGet, "/").Build()
			mw := SecurityHeaders(config.SecurityHeadersConfig{
				ContentSecurityPolicy:             tt.csp,
				ContentSecurityPolicyNonceEnabled: tt.nonceEnabled,
			})
			w := zhtest.TestMiddlewareWithHandler(mw, handler, req)

			zhtest.AssertWith(t, w).Status(http.StatusOK)

			csp := w.Header().Get(httpx.HeaderContentSecurityPolicy)
			if !strings.Contains(csp, tt.wantCSPContains) {
				t.Errorf("CSP header = %q, want containing %q", csp, tt.wantCSPContains)
			}

			if tt.wantNonce {
				if capturedNonce == "" {
					t.Error("Expected nonce in context, got empty string")
				}
				if !strings.Contains(csp, capturedNonce) {
					t.Errorf("CSP header %q does not contain nonce %q", csp, capturedNonce)
				}
			} else {
				if capturedNonce != "" {
					t.Errorf("Expected no nonce, got %q", capturedNonce)
				}
			}
		})
	}
}

func TestSecurityHeaders_CSPNonceReportOnly(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	mw := SecurityHeaders(config.SecurityHeadersConfig{
		ContentSecurityPolicy:             "script-src 'nonce-{{nonce}}'",
		ContentSecurityPolicyReportOnly:   true,
		ContentSecurityPolicyNonceEnabled: true,
	})
	w := zhtest.TestMiddlewareWithHandler(mw, handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderNotExists(httpx.HeaderContentSecurityPolicy)

	csp := w.Header().Get("Content-Security-Policy-Report-Only")
	if !strings.Contains(csp, "'nonce-") {
		t.Errorf("CSP-Report-Only header should contain nonce, got: %s", csp)
	}
}

func TestSecurityHeaders_CSPNonceCustomContextKey(t *testing.T) {
	type myNonceKey struct{}
	customKey := myNonceKey{}
	var capturedNonce string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedNonce = GetCSPNonce(r, customKey)
		w.WriteHeader(http.StatusOK)
	})

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	mw := SecurityHeaders(config.SecurityHeadersConfig{
		ContentSecurityPolicy:                "script-src 'nonce-{{nonce}}'",
		ContentSecurityPolicyNonceEnabled:    true,
		ContentSecurityPolicyNonceContextKey: customKey,
	})
	w := zhtest.TestMiddlewareWithHandler(mw, handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)

	if capturedNonce == "" {
		t.Error("Expected nonce with custom context key, got empty string")
	}
}

func TestGetCSPNonce_NotFound(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	nonce := GetCSPNonce(req)
	if nonce != "" {
		t.Errorf("Expected empty string for missing nonce, got %q", nonce)
	}
}

package middleware

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
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
		SecurityHeaders(
			config.WithSecurityHeadersHSTS(
				config.WithHSTSMaxAge(31536000),
				config.WithHSTSPreload(true),
			),
		),
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
		SecurityHeaders(
			config.WithSecurityHeadersCSP("default-src 'self'"),
			config.WithSecurityHeadersExemptPaths([]string{"/skipme"}),
		),
		handler,
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderNotExists("Content-Security-Policy")
}

func TestSecurityHeaders_DefaultValuesFill(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(SecurityHeaders(), req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).
		HeaderExists("Content-Security-Policy").
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
		SecurityHeaders(config.WithSecurityHeadersServer("")),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderNotExists("Server")
}

func TestSecurityHeaders_ContentSecurityPolicyNotSet(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SecurityHeaders(config.WithSecurityHeadersCSP("")),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).Header("Content-Security-Policy", config.DefaultSecurityHeadersConfig.ContentSecurityPolicy)
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
			req := zhtest.NewRequest(http.MethodGet, "/").Build()
			w := zhtest.TestMiddlewareWithHandler(tt.option, handler, req)

			zhtest.AssertWith(t, w).Status(http.StatusOK)
			zhtest.AssertWith(t, w).Header(tt.header, tt.expected)
		})
	}
}

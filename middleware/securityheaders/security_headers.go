package securityheaders

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
)

// New creates a security headers middleware with the provided configuration
// that adds various security-related HTTP headers to responses
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "SecurityHeaders")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			csp := c.ContentSecurityPolicy

			// Generate CSP nonce if enabled and placeholder is present
			if c.ContentSecurityPolicyNonceEnabled && strings.Contains(csp, CSPNoncePlaceholder) {
				nonce := generateNonce()
				csp = strings.ReplaceAll(csp, CSPNoncePlaceholder, nonce)

				// Store nonce in context for handlers to use
				ctxKey := c.ContentSecurityPolicyNonceContextKey
				if ctxKey == nil {
					ctxKey = CSPNonceContextKey
				}
				r = r.WithContext(context.WithValue(r.Context(), ctxKey, nonce))
			}

			if csp != "" {
				if c.ContentSecurityPolicyReportOnly {
					w.Header().Set(httpx.HeaderContentSecurityPolicyReportOnly, csp)
				} else {
					w.Header().Set(httpx.HeaderContentSecurityPolicy, csp)
				}
			}

			if c.CrossOriginEmbedderPolicy != "" {
				w.Header().Set(httpx.HeaderCrossOriginEmbedderPolicy, c.CrossOriginEmbedderPolicy)
			}
			if c.CrossOriginOpenerPolicy != "" {
				w.Header().Set(httpx.HeaderCrossOriginOpenerPolicy, c.CrossOriginOpenerPolicy)
			}
			if c.CrossOriginResourcePolicy != "" {
				w.Header().Set(httpx.HeaderCrossOriginResourcePolicy, c.CrossOriginResourcePolicy)
			}

			if c.PermissionsPolicy != "" {
				w.Header().Set(httpx.HeaderPermissionsPolicy, c.PermissionsPolicy)
			}

			if c.ReferrerPolicy != "" {
				w.Header().Set(httpx.HeaderReferrerPolicy, c.ReferrerPolicy)
			}

			if c.Server != "" {
				w.Header().Set(httpx.HeaderServer, c.Server)
			}

			// HSTS (only for HTTPS requests)
			if c.StrictTransportSecurity.MaxAge != 0 && isHTTPS(r) {
				hstsValue := fmt.Sprintf("max-age=%d", c.StrictTransportSecurity.MaxAge)
				if !c.StrictTransportSecurity.ExcludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				if c.StrictTransportSecurity.PreloadEnabled {
					hstsValue += "; preload"
				}
				w.Header().Set(httpx.HeaderStrictTransportSecurity, hstsValue)
			}

			if c.XContentTypeOptions != "" {
				w.Header().Set(httpx.HeaderXContentTypeOptions, c.XContentTypeOptions)
			}

			if c.XFrameOptions != "" {
				w.Header().Set(httpx.HeaderXFrameOptions, c.XFrameOptions)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isHTTPS checks if the request is over HTTPS
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil ||
		r.Header.Get(httpx.HeaderXForwardedProto) == "https" ||
		r.Header.Get(httpx.HeaderXForwardedProtocol) == "https"
}

// generateNonce creates a random base64-encoded nonce for CSP
func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// GetCSPNonce retrieves the CSP nonce from the request context.
// Returns empty string if nonce is not present.
func GetCSPNonce(r *http.Request, key ...any) string {
	var ctxKey any
	if len(key) > 0 {
		ctxKey = key[0]
	} else {
		ctxKey = CSPNonceContextKey
	}
	if nonce, ok := r.Context().Value(ctxKey).(string); ok {
		return nonce
	}
	return ""
}

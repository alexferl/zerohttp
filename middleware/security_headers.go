package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	zconfig "github.com/alexferl/zerohttp/internal/config"

	"github.com/alexferl/zerohttp/config"
)

// SecurityHeaders creates a security headers middleware that adds various security-related HTTP headers
func SecurityHeaders(cfg ...config.SecurityHeadersConfig) func(http.Handler) http.Handler {
	c := config.DefaultSecurityHeadersConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			csp := c.ContentSecurityPolicy

			// Generate CSP nonce if enabled and placeholder is present
			if c.ContentSecurityPolicyNonceEnabled && strings.Contains(csp, config.CSPNoncePlaceholder) {
				nonce := generateNonce()
				csp = strings.ReplaceAll(csp, config.CSPNoncePlaceholder, nonce)

				// Store nonce in context for handlers to use
				ctxKey := c.ContentSecurityPolicyNonceContextKey
				if ctxKey == "" {
					ctxKey = config.DefaultCSPNonceContextKey
				}
				r = r.WithContext(context.WithValue(r.Context(), ctxKey, nonce))
			}

			if csp != "" {
				if c.ContentSecurityPolicyReportOnly {
					w.Header().Set("Content-Security-Policy-Report-Only", csp)
				} else {
					w.Header().Set("Content-Security-Policy", csp)
				}
			}

			if c.CrossOriginEmbedderPolicy != "" {
				w.Header().Set("Cross-Origin-Embedder-Policy", c.CrossOriginEmbedderPolicy)
			}
			if c.CrossOriginOpenerPolicy != "" {
				w.Header().Set("Cross-Origin-Opener-Policy", c.CrossOriginOpenerPolicy)
			}
			if c.CrossOriginResourcePolicy != "" {
				w.Header().Set("Cross-Origin-Resource-Policy", c.CrossOriginResourcePolicy)
			}

			if c.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", c.PermissionsPolicy)
			}

			if c.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", c.ReferrerPolicy)
			}

			if c.Server != "" {
				w.Header().Set("Server", c.Server)
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
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			if c.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", c.XContentTypeOptions)
			}

			if c.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", c.XFrameOptions)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isHTTPS checks if the request is over HTTPS
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil ||
		r.Header.Get("X-Forwarded-Proto") == "https" ||
		r.Header.Get("X-Forwarded-Protocol") == "https"
}

// generateNonce creates a random base64-encoded nonce for CSP
func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// GetCSPNonce retrieves the CSP nonce from the request context.
// Returns empty string if nonce is not present.
func GetCSPNonce(r *http.Request, key ...config.CSPNonceContextKey) string {
	var ctxKey any
	if len(key) > 0 {
		ctxKey = key[0]
	} else {
		ctxKey = config.DefaultCSPNonceContextKey
	}
	if nonce, ok := r.Context().Value(ctxKey).(string); ok {
		return nonce
	}
	return ""
}

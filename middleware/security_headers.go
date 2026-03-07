package middleware

import (
	"fmt"
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// SecurityHeaders creates a security headers middleware that adds various security-related HTTP headers
func SecurityHeaders(opts ...config.SecurityHeadersOption) func(http.Handler) http.Handler {
	cfg := config.DefaultSecurityHeadersConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.ContentSecurityPolicy == "" {
		cfg.ContentSecurityPolicy = config.DefaultSecurityHeadersConfig.ContentSecurityPolicy
	}
	if cfg.CrossOriginEmbedderPolicy == "" {
		cfg.CrossOriginEmbedderPolicy = config.DefaultSecurityHeadersConfig.CrossOriginEmbedderPolicy
	}
	if cfg.CrossOriginOpenerPolicy == "" {
		cfg.CrossOriginOpenerPolicy = config.DefaultSecurityHeadersConfig.CrossOriginOpenerPolicy
	}
	if cfg.CrossOriginResourcePolicy == "" {
		cfg.CrossOriginResourcePolicy = config.DefaultSecurityHeadersConfig.CrossOriginResourcePolicy
	}
	if cfg.PermissionsPolicy == "" {
		cfg.PermissionsPolicy = config.DefaultSecurityHeadersConfig.PermissionsPolicy
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = config.DefaultSecurityHeadersConfig.ReferrerPolicy
	}
	if cfg.XContentTypeOptions == "" {
		cfg.XContentTypeOptions = config.DefaultSecurityHeadersConfig.XContentTypeOptions
	}
	if cfg.XFrameOptions == "" {
		cfg.XFrameOptions = config.DefaultSecurityHeadersConfig.XFrameOptions
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if r.URL.Path == exemptPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			if cfg.ContentSecurityPolicy != "" {
				if cfg.ContentSecurityPolicyReportOnly {
					w.Header().Set("Content-Security-Policy-Report-Only", cfg.ContentSecurityPolicy)
				} else {
					w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
				}
			}

			if cfg.CrossOriginEmbedderPolicy != "" {
				w.Header().Set("Cross-Origin-Embedder-Policy", cfg.CrossOriginEmbedderPolicy)
			}
			if cfg.CrossOriginOpenerPolicy != "" {
				w.Header().Set("Cross-Origin-Opener-Policy", cfg.CrossOriginOpenerPolicy)
			}
			if cfg.CrossOriginResourcePolicy != "" {
				w.Header().Set("Cross-Origin-Resource-Policy", cfg.CrossOriginResourcePolicy)
			}

			if cfg.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			if cfg.Server != "" {
				w.Header().Set("Server", cfg.Server)
			}

			// HSTS (only for HTTPS requests)
			if cfg.StrictTransportSecurity.MaxAge != 0 && isHTTPS(r) {
				hstsValue := fmt.Sprintf("max-age=%d", cfg.StrictTransportSecurity.MaxAge)
				if !cfg.StrictTransportSecurity.ExcludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				if cfg.StrictTransportSecurity.PreloadEnabled {
					hstsValue += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			if cfg.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", cfg.XContentTypeOptions)
			}

			if cfg.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
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

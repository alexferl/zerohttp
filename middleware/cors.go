package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(opts ...config.CORSOption) func(http.Handler) http.Handler {
	cfg := config.DefaultCORSConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	// Set defaults if not provided
	if cfg.AllowedOrigins == nil {
		cfg.AllowedOrigins = config.DefaultCORSConfig.AllowedOrigins
	}
	if cfg.AllowedMethods == nil {
		cfg.AllowedMethods = config.DefaultCORSConfig.AllowedMethods
	}
	if cfg.AllowedHeaders == nil {
		cfg.AllowedHeaders = config.DefaultCORSConfig.AllowedHeaders
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = config.DefaultCORSConfig.MaxAge
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultCORSConfig.ExemptPaths
	}

	allowedOriginMap := make(map[string]bool)
	allowAllOrigins := false
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
		allowedOriginMap[strings.ToLower(origin)] = true
	}

	allowedMethodMap := make(map[string]bool)
	for _, method := range cfg.AllowedMethods {
		allowedMethodMap[strings.ToUpper(method)] = true
	}

	allowedHeaderMap := make(map[string]bool)
	for _, header := range cfg.AllowedHeaders {
		allowedHeaderMap[strings.ToLower(header)] = true
	}

	allowedMethodsHeader := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeadersHeader := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeadersHeader := strings.Join(cfg.ExposedHeaders, ", ")
	maxAgeHeader := strconv.Itoa(cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			origin := r.Header.Get("Origin")

			// Only process CORS if Origin header is present
			if origin == "" {
				// No origin header means this is not a cross-origin request
				if r.Method == http.MethodOptions && cfg.OptionsPassthrough {
					next.ServeHTTP(w, r)
					return
				} else if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Determine if origin is allowed
			var allowedOrigin string
			if allowAllOrigins {
				if cfg.AllowCredentials {
					// When credentials are allowed, can't use "*"
					allowedOrigin = origin
				} else {
					allowedOrigin = "*"
				}
			} else if allowedOriginMap[strings.ToLower(origin)] {
				allowedOrigin = origin
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if len(cfg.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", exposedHeadersHeader)
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				if allowedOrigin == "" {
					// Origin not allowed, don't set preflight headers
					if cfg.OptionsPassthrough {
						next.ServeHTTP(w, r)
						return
					}
					w.WriteHeader(http.StatusNoContent)
					return
				}

				// Check if requested method is allowed
				requestMethod := r.Header.Get("Access-Control-Request-Method")
				if requestMethod != "" && !allowedMethodMap[strings.ToUpper(requestMethod)] {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				// Check if requested headers are allowed
				requestHeaders := r.Header.Get("Access-Control-Request-Headers")
				if requestHeaders != "" {
					headers := strings.Split(requestHeaders, ",")
					for _, header := range headers {
						header = strings.ToLower(strings.TrimSpace(header))
						if !allowedHeaderMap[header] {
							w.WriteHeader(http.StatusForbidden)
							return
						}
					}
				}

				// Set preflight response headers
				w.Header().Set("Access-Control-Allow-Methods", allowedMethodsHeader)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeadersHeader)
				w.Header().Set("Access-Control-Max-Age", maxAgeHeader)

				if cfg.OptionsPassthrough {
					next.ServeHTTP(w, r)
					return
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			// For actual requests, just continue
			next.ServeHTTP(w, r)
		})
	}
}

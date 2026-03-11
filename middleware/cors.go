package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(cfg ...config.CORSConfig) func(http.Handler) http.Handler {
	c := config.DefaultCORSConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	// Set defaults if not provided
	if c.AllowedOrigins == nil {
		c.AllowedOrigins = config.DefaultCORSConfig.AllowedOrigins
	}
	if c.AllowedMethods == nil {
		c.AllowedMethods = config.DefaultCORSConfig.AllowedMethods
	}
	if c.AllowedHeaders == nil {
		c.AllowedHeaders = config.DefaultCORSConfig.AllowedHeaders
	}
	if c.MaxAge == 0 {
		c.MaxAge = config.DefaultCORSConfig.MaxAge
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultCORSConfig.ExemptPaths
	}

	allowedOriginMap := make(map[string]bool)
	allowAllOrigins := false
	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
		allowedOriginMap[strings.ToLower(origin)] = true
	}

	allowedMethodMap := make(map[string]bool)
	for _, method := range c.AllowedMethods {
		allowedMethodMap[strings.ToUpper(method)] = true
	}

	allowedHeaderMap := make(map[string]bool)
	for _, header := range c.AllowedHeaders {
		allowedHeaderMap[strings.ToLower(header)] = true
	}

	allowedMethodsHeader := strings.Join(c.AllowedMethods, ", ")
	allowedHeadersHeader := strings.Join(c.AllowedHeaders, ", ")
	exposedHeadersHeader := strings.Join(c.ExposedHeaders, ", ")
	maxAgeHeader := strconv.Itoa(c.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			origin := r.Header.Get("Origin")

			// Only process CORS if Origin header is present
			if origin == "" {
				// No origin header means this is not a cross-origin request
				if r.Method == http.MethodOptions && c.OptionsPassthrough {
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
			originAllowed := false

			if c.AllowOriginFunc != nil {
				// Use custom origin validator
				originAllowed = c.AllowOriginFunc(origin)
				// Set Vary header when using dynamic origin validation
				w.Header().Set("Vary", "Origin")
			} else if allowAllOrigins {
				originAllowed = true
			} else if allowedOriginMap[strings.ToLower(origin)] {
				originAllowed = true
			}

			if originAllowed {
				if allowAllOrigins && !c.AllowCredentials {
					// When credentials are allowed, can't use "*"
					allowedOrigin = "*"
				} else {
					allowedOrigin = origin
				}
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)

				if c.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if len(c.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", exposedHeadersHeader)
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				// Record preflight request metric
				reg.Counter("cors_preflight_requests_total").Inc()

				if allowedOrigin == "" {
					// Origin not allowed, don't set preflight headers
					reg.Counter("cors_requests_total", "origin").WithLabelValues("rejected").Inc()
					if c.OptionsPassthrough {
						next.ServeHTTP(w, r)
						return
					}
					w.WriteHeader(http.StatusNoContent)
					return
				}

				// Check if requested method is allowed
				requestMethod := r.Header.Get("Access-Control-Request-Method")
				if requestMethod != "" && !allowedMethodMap[strings.ToUpper(requestMethod)] {
					detail := problem.NewDetail(http.StatusMethodNotAllowed, "Method not allowed")
					w.Header().Set("Allow", allowedMethodsHeader)
					_ = detail.Render(w)
					return
				}

				// Check if requested headers are allowed
				requestHeaders := r.Header.Get("Access-Control-Request-Headers")
				if requestHeaders != "" {
					headers := strings.Split(requestHeaders, ",")
					for _, header := range headers {
						header = strings.ToLower(strings.TrimSpace(header))
						if !allowedHeaderMap[header] {
							detail := problem.NewDetail(http.StatusForbidden, "Request header not allowed")
							_ = detail.Render(w)
							return
						}
					}
				}

				// Set preflight response headers
				w.Header().Set("Access-Control-Allow-Methods", allowedMethodsHeader)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeadersHeader)
				w.Header().Set("Access-Control-Max-Age", maxAgeHeader)

				if c.OptionsPassthrough {
					next.ServeHTTP(w, r)
					return
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			// For actual requests, record origin result
			if origin != "" {
				result := "allowed"
				if allowedOrigin == "" {
					result = "rejected"
				}
				reg.Counter("cors_requests_total", "origin").WithLabelValues(result).Inc()
			}

			next.ServeHTTP(w, r)
		})
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// ContentEncoding enforces a whitelist of request Content-Encoding
func ContentEncoding(cfg ...config.ContentEncodingConfig) func(http.Handler) http.Handler {
	c := config.DefaultContentEncodingConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.Encodings == nil {
		c.Encodings = config.DefaultContentEncodingConfig.Encodings
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultContentEncodingConfig.ExemptPaths
	}

	allowedEncodings := make(map[string]struct{}, len(c.Encodings))
	for _, encoding := range c.Encodings {
		allowedEncodings[strings.TrimSpace(strings.ToLower(encoding))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check exempt paths
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Skip validation for empty content body (like Chi does)
			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Validate all Content-Encoding headers
			for _, headerValue := range r.Header["Content-Encoding"] {
				encodings := strings.Split(headerValue, ",")
				for _, encoding := range encodings {
					encoding = strings.TrimSpace(strings.ToLower(encoding))
					if encoding != "" {
						if _, ok := allowedEncodings[encoding]; !ok {
							w.WriteHeader(http.StatusUnsupportedMediaType)
							return
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

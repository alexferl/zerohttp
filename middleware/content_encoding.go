package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// ContentEncoding enforces a whitelist of request Content-Encoding
func ContentEncoding(opts ...config.ContentEncodingOption) func(http.Handler) http.Handler {
	cfg := config.DefaultContentEncodingConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Encodings == nil {
		cfg.Encodings = config.DefaultContentEncodingConfig.Encodings
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultContentEncodingConfig.ExemptPaths
	}

	allowedEncodings := make(map[string]struct{}, len(cfg.Encodings))
	for _, encoding := range cfg.Encodings {
		allowedEncodings[strings.TrimSpace(strings.ToLower(encoding))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check exempt paths
			for _, exemptPath := range cfg.ExemptPaths {
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

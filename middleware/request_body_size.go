package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// RequestBodySize creates a request size limiting middleware with optional configuration
func RequestBodySize(opts ...config.RequestBodySizeOption) func(http.Handler) http.Handler {
	cfg := config.DefaultRequestBodySizeConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = config.DefaultRequestBodySizeConfig.MaxBytes
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultRequestBodySizeConfig.ExemptPaths
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

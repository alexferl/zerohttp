package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// RequestBodySize creates a request size limiting middleware with the provided configuration
func RequestBodySize(cfg ...config.RequestBodySizeConfig) func(http.Handler) http.Handler {
	c := config.DefaultRequestBodySizeConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = config.DefaultRequestBodySizeConfig.MaxBytes
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultRequestBodySizeConfig.ExemptPaths
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			r.Body = http.MaxBytesReader(w, r.Body, c.MaxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

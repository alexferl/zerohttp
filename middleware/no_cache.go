package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// NoCache middleware sets headers on every response to prevent caching and deletes ETag headers.
func NoCache(opts ...config.NoCacheOption) func(http.Handler) http.Handler {
	cfg := config.DefaultNoCacheConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.NoCacheHeaders == nil {
		cfg.NoCacheHeaders = config.DefaultNoCacheHeaders
	}
	if cfg.ETagHeaders == nil {
		cfg.ETagHeaders = config.DefaultETagHeaders
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Delete ETag/conditional headers in request
			for _, v := range cfg.ETagHeaders {
				if r.Header.Get(v) != "" {
					r.Header.Del(v)
				}
			}

			// Set no-cache headers on response
			for k, v := range cfg.NoCacheHeaders {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}

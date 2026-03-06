package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// NoCache middleware sets headers on every response to prevent caching and deletes ETag headers.
func NoCache(cfg ...config.NoCacheConfig) func(http.Handler) http.Handler {
	c := config.DefaultNoCacheConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.NoCacheHeaders == nil {
		c.NoCacheHeaders = config.DefaultNoCacheHeaders
	}
	if c.ETagHeaders == nil {
		c.ETagHeaders = config.DefaultETagHeaders
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Delete ETag/conditional headers in request
			for _, v := range c.ETagHeaders {
				if r.Header.Get(v) != "" {
					r.Header.Del(v)
				}
			}

			// Set no-cache headers on response
			for k, v := range c.NoCacheHeaders {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}

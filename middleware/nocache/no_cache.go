package nocache

import (
	"net/http"

	zconfig "github.com/alexferl/zerohttp/internal/config"
)

// New creates a no-cache middleware with the provided configuration that sets headers
// on every response to prevent caching and deletes ETag headers.
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
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
			for k, v := range c.Headers {
				w.Header().Set(k, v)
			}

			next.ServeHTTP(w, r)
		})
	}
}

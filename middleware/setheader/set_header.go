package setheader

import (
	"net/http"

	zconfig "github.com/alexferl/zerohttp/internal/config"
)

// New creates a set header middleware with the provided configuration that sets response headers
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[len(cfg)-1])
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for key, value := range c.Headers {
				w.Header().Set(key, value)
			}
			next.ServeHTTP(w, r)
		})
	}
}

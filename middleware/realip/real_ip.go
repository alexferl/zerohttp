package realip

import (
	"net"
	"net/http"

	zconfig "github.com/alexferl/zerohttp/internal/config"
)

// New creates a real IP middleware with the provided configuration that sets
// r.RemoteAddr to the extracted real client IP.
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	if c.IPExtractor == nil {
		c.IPExtractor = DefaultConfig.IPExtractor
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := c.IPExtractor(r)
			// Set RemoteAddr to real IP, preserving the port for consistency
			_, port, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil && port != "" {
				r.RemoteAddr = net.JoinHostPort(realIP, port)
			} else {
				r.RemoteAddr = realIP
			}
			next.ServeHTTP(w, r)
		})
	}
}

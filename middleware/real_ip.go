package middleware

import (
	"net"
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// RealIP middleware sets r.RemoteAddr to the extracted real client IP.
func RealIP(cfg ...config.RealIPConfig) func(http.Handler) http.Handler {
	c := config.DefaultRealIPConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.IPExtractor == nil {
		c.IPExtractor = config.DefaultRealIPConfig.IPExtractor
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

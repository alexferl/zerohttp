package middleware

import (
	"net"
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// RealIP middleware sets r.RemoteAddr to the extracted real client IP.
func RealIP(opts ...config.RealIPOption) func(http.Handler) http.Handler {
	cfg := config.DefaultRealIPConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.IPExtractor == nil {
		cfg.IPExtractor = config.DefaultRealIPConfig.IPExtractor
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := cfg.IPExtractor(r)
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

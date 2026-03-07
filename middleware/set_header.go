package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// SetHeader is a middleware that sets response headers
func SetHeader(opts ...config.SetHeaderOption) func(http.Handler) http.Handler {
	cfg := config.DefaultSetHeaderConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for key, value := range cfg.Headers {
				w.Header().Set(key, value)
			}
			next.ServeHTTP(w, r)
		})
	}
}

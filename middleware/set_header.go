package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// SetHeader is a middleware that sets response headers
func SetHeader(cfg ...config.SetHeaderConfig) func(http.Handler) http.Handler {
	c := config.DefaultSetHeaderConfig
	if len(cfg) > 0 {
		c.Headers = cfg[len(cfg)-1].Headers
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

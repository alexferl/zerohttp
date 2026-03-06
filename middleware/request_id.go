package middleware

import (
	"context"
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// RequestID creates a request ID middleware with the provided configuration
func RequestID(cfg ...config.RequestIDConfig) func(http.Handler) http.Handler {
	c := config.DefaultRequestIDConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Header == "" {
		c.Header = config.DefaultRequestIDConfig.Header
	}
	if c.Generator == nil {
		c.Generator = config.DefaultRequestIDConfig.Generator
	}
	if c.ContextKey == "" {
		c.ContextKey = config.DefaultRequestIDConfig.ContextKey
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(c.Header)

			if requestID == "" {
				requestID = c.Generator()
				r.Header.Set(c.Header, requestID)
			}

			w.Header().Set(c.Header, requestID)

			ctx := context.WithValue(r.Context(), c.ContextKey, requestID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID retrieves the request ID from context using the specified key
// If no key is provided, uses the default key
func GetRequestID(ctx context.Context, key ...config.RequestIDContextKey) string {
	var contextKey config.RequestIDContextKey
	if len(key) > 0 {
		contextKey = key[0]
	} else {
		contextKey = config.DefaultRequestIDConfig.ContextKey
	}

	if requestID, ok := ctx.Value(contextKey).(string); ok {
		return requestID
	}
	return ""
}

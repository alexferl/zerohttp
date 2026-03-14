package middleware

import (
	"context"
	"net/http"

	"github.com/alexferl/zerohttp/config"
	zconfig "github.com/alexferl/zerohttp/internal/config"
)

// RequestID creates a request ID middleware with the provided configuration
func RequestID(cfg ...config.RequestIDConfig) func(http.Handler) http.Handler {
	c := config.DefaultRequestIDConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
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
func GetRequestID(ctx context.Context, key ...any) string {
	var contextKey any
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

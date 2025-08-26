package middleware

import (
	"context"
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// RequestID creates a request ID middleware with optional configuration
func RequestID(opts ...config.RequestIDOption) func(http.Handler) http.Handler {
	cfg := config.DefaultRequestIDConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Header == "" {
		cfg.Header = config.DefaultRequestIDConfig.Header
	}
	if cfg.Generator == nil {
		cfg.Generator = config.DefaultRequestIDConfig.Generator
	}
	if cfg.ContextKey == "" {
		cfg.ContextKey = config.DefaultRequestIDConfig.ContextKey
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(cfg.Header)

			if requestID == "" {
				requestID = cfg.Generator()
				r.Header.Set(cfg.Header, requestID)
			}

			w.Header().Set(cfg.Header, requestID)

			ctx := context.WithValue(r.Context(), cfg.ContextKey, requestID)
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

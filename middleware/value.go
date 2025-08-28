package middleware

import (
	"context"
	"net/http"
)

// WithValue sets a key/value pair in the request context for downstream handlers.
func WithValue(key, val any) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), key, val))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// GetContextValue retrieves a typed value from the request context.
// Returns the value and true if found and correctly typed, zero value and false otherwise.
func GetContextValue[T any](r *http.Request, key any) (T, bool) {
	val := r.Context().Value(key)
	if val == nil {
		var zero T
		return zero, false
	}

	if typed, ok := val.(T); ok {
		return typed, true
	}
	var zero T

	return zero, false
}

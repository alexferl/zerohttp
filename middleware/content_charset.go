package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// ContentCharset generates a middleware that validates request charset and returns
// 415 Unsupported Media Type if the charset doesn't match the allowed list
func ContentCharset(opts ...config.ContentCharsetOption) func(http.Handler) http.Handler {
	cfg := config.DefaultContentCharsetConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Charsets == nil {
		cfg.Charsets = config.DefaultContentCharsetConfig.Charsets
	}

	// Normalize charsets to lowercase
	normalizedCharsets := make([]string, len(cfg.Charsets))
	for i, c := range cfg.Charsets {
		normalizedCharsets[i] = strings.ToLower(c)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !contentEncoding(r.Header.Get("Content-Type"), normalizedCharsets...) {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Check the content encoding against a list of acceptable values
func contentEncoding(ce string, charsets ...string) bool {
	ce = strings.ToLower(ce)

	// Split on semicolons and check each part for charset
	parts := strings.Split(ce, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "charset") {
			// Handle "charset=value" or "charset = value"
			_, charset := split(part, "=")
			charset = strings.TrimSpace(charset)

			// Check if this charset is allowed
			for _, c := range charsets {
				if charset == c {
					return true
				}
			}
			return false // Found charset but not allowed
		}
	}

	// No charset found, check if empty is allowed
	for _, c := range charsets {
		if c == "" {
			return true
		}
	}
	return false
}

// Split a string in two parts, cleaning any whitespace
func split(str, sep string) (string, string) {
	a, b, found := strings.Cut(str, sep)
	a = strings.TrimSpace(a)
	if found {
		b = strings.TrimSpace(b)
	}
	return a, b
}

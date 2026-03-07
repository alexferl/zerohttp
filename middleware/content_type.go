package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// ContentType enforces a whitelist of request Content-Types otherwise responds
// with a 415 Unsupported Media Type status.
func ContentType(opts ...config.ContentTypeOption) func(http.Handler) http.Handler {
	cfg := config.DefaultContentTypeConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.ContentTypes == nil {
		cfg.ContentTypes = config.DefaultContentTypeConfig.ContentTypes
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultContentTypeConfig.ExemptPaths
	}

	allowedContentTypes := make(map[string]struct{}, len(cfg.ContentTypes))
	for _, ctype := range cfg.ContentTypes {
		allowedContentTypes[strings.TrimSpace(strings.ToLower(ctype))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Extract content type without parameters (charset, boundary, etc.)
			contentType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
			contentType = strings.ToLower(strings.TrimSpace(contentType))

			if _, ok := allowedContentTypes[contentType]; !ok {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

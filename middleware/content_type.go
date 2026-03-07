package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// ContentType enforces a whitelist of request Content-Types otherwise responds
// with a 415 Unsupported Media Type status.
func ContentType(cfg ...config.ContentTypeConfig) func(http.Handler) http.Handler {
	c := config.DefaultContentTypeConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.ContentTypes == nil {
		c.ContentTypes = config.DefaultContentTypeConfig.ContentTypes
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultContentTypeConfig.ExemptPaths
	}

	allowedContentTypes := make(map[string]struct{}, len(c.ContentTypes))
	for _, ctype := range c.ContentTypes {
		allowedContentTypes[strings.TrimSpace(strings.ToLower(ctype))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
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

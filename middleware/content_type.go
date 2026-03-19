package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/problem"
)

// ContentType enforces a whitelist of request Content-Types otherwise responds
// with a 415 Unsupported Media Type status.
func ContentType(cfg ...config.ContentTypeConfig) func(http.Handler) http.Handler {
	c := config.DefaultContentTypeConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	validatePathConfig(c.ExcludedPaths, c.IncludedPaths, "ContentType")

	allowedContentTypes := make(map[string]struct{}, len(c.ContentTypes))
	for _, ctype := range c.ContentTypes {
		allowedContentTypes[strings.TrimSpace(strings.ToLower(ctype))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Extract content type without parameters (charset, boundary, etc.)
			contentType, _, _ := strings.Cut(r.Header.Get(httpx.HeaderContentType), ";")
			contentType = strings.ToLower(strings.TrimSpace(contentType))

			if _, ok := allowedContentTypes[contentType]; !ok {
				detail := problem.NewDetail(http.StatusUnsupportedMediaType, "Unsupported content type")
				if len(c.ContentTypes) > 0 {
					w.Header().Set(httpx.HeaderAcceptPost, c.ContentTypes[0])
				}
				_ = detail.RenderAuto(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

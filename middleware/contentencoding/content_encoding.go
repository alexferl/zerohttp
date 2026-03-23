package contentencoding

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/internal/problem"
)

// New creates a content encoding middleware with the provided configuration that enforces a whitelist of request Content-Encoding
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "ContentEncoding")

	allowedEncodings := make(map[string]struct{}, len(c.Encodings))
	for _, encoding := range c.Encodings {
		allowedEncodings[strings.TrimSpace(strings.ToLower(encoding))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			for _, headerValue := range r.Header[httpx.HeaderContentEncoding] {
				encodings := strings.Split(headerValue, ",")
				for _, encoding := range encodings {
					encoding = strings.TrimSpace(strings.ToLower(encoding))
					if encoding != "" {
						if _, ok := allowedEncodings[encoding]; !ok {
							detail := problem.NewDetail(http.StatusUnsupportedMediaType, "Unsupported content encoding")
							if len(c.Encodings) > 0 {
								w.Header().Set(httpx.HeaderAcceptEncoding, strings.Join(c.Encodings, ", "))
							}
							_ = detail.RenderAuto(w, r)
							return
						}
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

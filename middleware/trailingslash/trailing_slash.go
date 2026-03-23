package trailingslash

import (
	"net/http"
	"strings"

	zconfig "github.com/alexferl/zerohttp/internal/config"
)

// New creates a trailing slash middleware with the provided configuration that handles
// trailing slashes in URLs.
//
// IMPORTANT: Register routes WITHOUT trailing slashes to use this middleware.
// If you register "/docs/", Go's ServeMux auto-redirects "/docs" before
// middleware runs, bypassing this middleware entirely.
//
// Good:  router.GET("/docs", handler)  // middleware handles the redirect
// Bad:   router.GET("/docs/", handler) // ServeMux handles the redirect
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Don't modify root path
			if path == "/" {
				next.ServeHTTP(w, r)
				return
			}

			hasTrailingSlash := strings.HasSuffix(path, "/")
			needsChange := false
			var newPath string

			if c.PreferTrailingSlash && !hasTrailingSlash {
				// Want trailing slash but don't have it
				needsChange = true
				newPath = path + "/"
			} else if !c.PreferTrailingSlash && hasTrailingSlash {
				// Don't want trailing slash but have it
				needsChange = true
				newPath = strings.TrimSuffix(path, "/")
			}

			if !needsChange {
				next.ServeHTTP(w, r)
				return
			}

			switch c.Action {
			case RedirectAction:
				// Build new URL with corrected path
				newURL := *r.URL
				newURL.Path = newPath
				http.Redirect(w, r, newURL.String(), c.RedirectCode)
				return

			case StripAction:
				if hasTrailingSlash {
					r.URL.Path = strings.TrimSuffix(path, "/")
				}
				next.ServeHTTP(w, r)
				return

			case AppendAction:
				if !hasTrailingSlash {
					r.URL.Path = path + "/"
				}
				next.ServeHTTP(w, r)
				return

			default:
				next.ServeHTTP(w, r)
				return
			}
		})
	}
}

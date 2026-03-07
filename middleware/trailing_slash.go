package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// TrailingSlash is a middleware that handles trailing slashes in URLs
func TrailingSlash(cfg ...config.TrailingSlashConfig) func(http.Handler) http.Handler {
	c := config.DefaultTrailingSlashConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.Action == "" {
		c.Action = config.DefaultTrailingSlashConfig.Action
	}
	if c.RedirectCode == 0 {
		c.RedirectCode = config.DefaultTrailingSlashConfig.RedirectCode
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
			case config.RedirectAction:
				// Build new URL with corrected path
				newURL := *r.URL
				newURL.Path = newPath
				http.Redirect(w, r, newURL.String(), c.RedirectCode)
				return

			case config.StripAction:
				if hasTrailingSlash {
					r.URL.Path = strings.TrimSuffix(path, "/")
				}
				next.ServeHTTP(w, r)
				return

			case config.AppendAction:
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

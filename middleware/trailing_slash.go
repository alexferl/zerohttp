package middleware

import (
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// TrailingSlash is a middleware that handles trailing slashes in URLs
func TrailingSlash(opts ...config.TrailingSlashOption) func(http.Handler) http.Handler {
	cfg := config.DefaultTrailingSlashConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Action == "" {
		cfg.Action = config.DefaultTrailingSlashConfig.Action
	}
	if cfg.RedirectCode == 0 {
		cfg.RedirectCode = config.DefaultTrailingSlashConfig.RedirectCode
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

			if cfg.PreferTrailingSlash && !hasTrailingSlash {
				// Want trailing slash but don't have it
				needsChange = true
				newPath = path + "/"
			} else if !cfg.PreferTrailingSlash && hasTrailingSlash {
				// Don't want trailing slash but have it
				needsChange = true
				newPath = strings.TrimSuffix(path, "/")
			}

			if !needsChange {
				next.ServeHTTP(w, r)
				return
			}

			switch cfg.Action {
			case config.RedirectAction:
				// Build new URL with corrected path
				newURL := *r.URL
				newURL.Path = newPath
				http.Redirect(w, r, newURL.String(), cfg.RedirectCode)
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

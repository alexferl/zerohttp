package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// BasicAuth creates a basic authentication middleware with optional configuration
func BasicAuth(opts ...config.BasicAuthOption) func(http.Handler) http.Handler {
	cfg := config.DefaultBasicAuthConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Realm == "" {
		cfg.Realm = config.DefaultBasicAuthConfig.Realm
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultBasicAuthConfig.ExemptPaths
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			user, pass, ok := r.BasicAuth()
			if !ok {
				basicAuthFailed(w, cfg.Realm)
				return
			}

			var isValid bool

			if cfg.Validator != nil {
				isValid = cfg.Validator(user, pass)
			} else if cfg.Credentials != nil {
				credPass, credUserOk := cfg.Credentials[user]
				isValid = credUserOk && subtle.ConstantTimeCompare([]byte(pass), []byte(credPass)) == 1
			} else {
				// No authentication configured - deny all
				isValid = false
			}

			if !isValid {
				basicAuthFailed(w, cfg.Realm)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
}

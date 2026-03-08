package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

// BasicAuth creates a basic authentication middleware with the provided configuration
func BasicAuth(cfg ...config.BasicAuthConfig) func(http.Handler) http.Handler {
	c := config.DefaultBasicAuthConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.Realm == "" {
		c.Realm = config.DefaultBasicAuthConfig.Realm
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultBasicAuthConfig.ExemptPaths
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			user, pass, ok := r.BasicAuth()
			if !ok {
				reg.Counter("basic_auth_requests_total", "result").WithLabelValues("missing").Inc()
				basicAuthFailed(w, c.Realm)
				return
			}

			var isValid bool

			if c.Validator != nil {
				isValid = c.Validator(user, pass)
			} else if c.Credentials != nil {
				credPass, credUserOk := c.Credentials[user]
				isValid = credUserOk && subtle.ConstantTimeCompare([]byte(pass), []byte(credPass)) == 1
			} else {
				isValid = false
			}

			if !isValid {
				reg.Counter("basic_auth_requests_total", "result").WithLabelValues("invalid").Inc()
				basicAuthFailed(w, c.Realm)
				return
			}

			reg.Counter("basic_auth_requests_total", "result").WithLabelValues("valid").Inc()
			next.ServeHTTP(w, r)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, realm string) {
	w.Header().Add("WWW-Authenticate", `Basic realm="`+realm+`"`)
	detail := problem.NewDetail(http.StatusUnauthorized, "Authentication required")
	_ = detail.Render(w)
}

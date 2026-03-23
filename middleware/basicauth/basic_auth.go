package basicauth

import (
	"crypto/subtle"
	"net/http"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

// New creates a basic authentication middleware with the provided configuration
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		config.Merge(&c, cfg[0])
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "BasicAuth")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			user, pass, ok := r.BasicAuth()
			if !ok {
				reg.Counter("basic_auth_requests_total", "result").WithLabelValues("missing").Inc()
				basicAuthFailed(w, r, c.Realm)
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
				basicAuthFailed(w, r, c.Realm)
				return
			}

			reg.Counter("basic_auth_requests_total", "result").WithLabelValues("valid").Inc()
			next.ServeHTTP(w, r)
		})
	}
}

func basicAuthFailed(w http.ResponseWriter, r *http.Request, realm string) {
	w.Header().Add(httpx.HeaderWWWAuthenticate, `Basic realm="`+realm+`"`)
	detail := problem.NewDetail(http.StatusUnauthorized, "Authentication required")
	_ = detail.RenderAuto(w, r)
}

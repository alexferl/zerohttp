package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/alexferl/zerohttp/config"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
)

// Recover is a middleware that recovers from panics, logs the panic (and a backtrace),
// and returns HTTP 500 if possible. It prints a request ID if one is provided.
//
// Note: Handler errors are handled directly by the router without panic.
// This middleware only catches actual panics from unexpected errors or explicit panic() calls.
func Recover(logger log.Logger, cfg ...config.RecoverConfig) func(http.Handler) http.Handler {
	c := config.DefaultRecoverConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}

					metrics.SafeRegistry(metrics.GetRegistry(r.Context())).Counter("recover_panics_total").Inc()

					// Real panic - log as error with stack trace
					reqID := r.Header.Get(c.RequestIDHeader)
					if reqID == "" {
						reqID = fmt.Sprintf("recover-%d", time.Now().UnixNano())
					}

					fields := []log.Field{
						log.P(rvr),
						log.F("request_id", reqID),
					}

					if c.EnableStackTrace {
						stack := make([]byte, c.StackSize)
						length := runtime.Stack(stack, false)
						fields = append(fields, log.F("stack", string(stack[:length])))
					}

					logger.Error("Recovered from panic", fields...)

					if r.Header.Get("Connection") != "Upgrade" {
						detail := problem.NewDetail(http.StatusInternalServerError, "Internal server error")
						_ = detail.Render(w)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

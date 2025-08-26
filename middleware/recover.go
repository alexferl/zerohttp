package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
)

// Recover is a middleware that recovers from panics, logs the panic (and a backtrace),
// and returns HTTP 500 if possible. It prints a request ID if one is provided.
func Recover(logger log.Logger, opts ...config.RecoverOption) func(http.Handler) http.Handler {
	cfg := config.DefaultRecoverConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.StackSize <= 0 {
		cfg.StackSize = config.DefaultRecoverConfig.StackSize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}

					reqID := r.Header.Get("X-Request-Id")
					if reqID == "" {
						reqID = fmt.Sprintf("recover-%d", time.Now().UnixNano())
					}

					fields := []log.Field{
						log.P(rvr),
						log.F("request_id", reqID),
					}

					if cfg.EnableStackTrace {
						stack := make([]byte, cfg.StackSize)
						length := runtime.Stack(stack, false)
						fields = append(fields, log.F("stack", string(stack[:length])))
					}

					logger.Error("Recovered from panic", fields...)

					if r.Header.Get("Connection") != "Upgrade" {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

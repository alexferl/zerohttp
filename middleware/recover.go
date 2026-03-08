package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/log"
)

// ValidationErrorer is the interface for validation errors.
// This is duplicated from the main package to avoid import cycle.
type ValidationErrorer interface {
	error
	ValidationErrors() map[string][]string
}

// Recover is a middleware that recovers from panics, logs the panic (and a backtrace),
// and returns HTTP 500 if possible. It prints a request ID if one is provided.
//
// It also handles expected errors from handlers:
//   - Validation errors (422 Unprocessable Entity)
//   - Binding errors (400 Bad Request)
func Recover(logger log.Logger, cfg ...config.RecoverConfig) func(http.Handler) http.Handler {
	c := config.DefaultRecoverConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}
	if c.StackSize <= 0 {
		c.StackSize = config.DefaultRecoverConfig.StackSize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					if rvr == http.ErrAbortHandler {
						panic(rvr)
					}

					// Check if this is a handler error (not a real panic)
					if err, ok := rvr.(error); ok {
						if unwrapped := unwrapHandlerError(err); unwrapped != nil {
							// Handle expected errors (validation, binding)
							handleExpectedError(w, r, logger, unwrapped)
							return
						}
					}

					// Real panic - log as error with stack trace
					reqID := r.Header.Get("X-Request-Id")
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
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// unwrapHandlerError checks if the error is a handler error wrapper
// and returns the underlying error.
func unwrapHandlerError(err error) error {
	// Handler errors are wrapped as "handler error: %w"
	if !strings.HasPrefix(err.Error(), "handler error: ") {
		return nil
	}
	return errors.Unwrap(err)
}

// handleExpectedError handles validation and binding errors
// by returning appropriate HTTP status codes without logging as ERROR.
func handleExpectedError(w http.ResponseWriter, _ *http.Request, logger log.Logger, err error) {
	// Check for validation errors (422)
	var verr ValidationErrorer
	if errors.As(err, &verr) {
		detail := problem.NewDetail(http.StatusUnprocessableEntity, "Validation failed")
		detail.Set("errors", verr.ValidationErrors())
		_ = detail.Render(w)
		return
	}

	// Check for binding errors (400)
	// bindError prefix: "bind error: "
	if strings.HasPrefix(err.Error(), "bind error: ") {
		// Log the actual error for debugging, but return a sanitized message
		logger.Debug("Binding error", log.P(err))

		detail := problem.NewDetail(http.StatusBadRequest, "Invalid request body")
		_ = detail.Render(w)
		return
	}

	// Unknown error type - treat as 500
	logger.Error("Unexpected handler error", log.P(err))
	detail := problem.NewDetail(http.StatusInternalServerError, "Internal server error")
	_ = detail.Render(w)
}

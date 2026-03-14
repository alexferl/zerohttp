package middleware

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/alexferl/zerohttp/config"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

// ErrTimeoutWrite is returned when the timeout middleware fails to write response data.
var ErrTimeoutWrite = errors.New("zerohttp: timeout middleware write failed")

// Timeout is a middleware that enforces request timeouts by canceling the context
// after a specified duration. When the timeout is exceeded, it returns an HTTP 504
// Gateway Timeout response to the client.
//
// Important: Your handler must monitor the ctx.Done() channel to detect when the
// context deadline has been reached. If you don't check this channel and return
// appropriately, the timeout mechanism will be ineffective and the request will
// continue processing beyond the intended timeout period.
func Timeout(cfg ...config.TimeoutConfig) func(http.Handler) http.Handler {
	c := config.DefaultTimeoutConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ctx, cancel := context.WithTimeout(r.Context(), c.Timeout)
			defer cancel()

			done := make(chan struct{})
			panicChan := make(chan any, 1)

			tw := &timeoutWriter{
				w:   w,
				h:   make(http.Header),
				req: r,
			}

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()
				next.ServeHTTP(tw, r.WithContext(ctx))
				close(done)
			}()

			select {
			case p := <-panicChan:
				panic(p)
			case <-done:
				tw.mu.Lock()
				defer tw.mu.Unlock()

				dst := w.Header()
				for k, v := range tw.h {
					dst[k] = v
				}

				if !tw.wroteHeader {
					tw.code = http.StatusOK
				}
				w.WriteHeader(tw.code)
				_, _ = w.Write(tw.wbuf.Bytes()) // Best effort write
			case <-ctx.Done():
				tw.mu.Lock()
				defer tw.mu.Unlock()

				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					metrics.SafeRegistry(metrics.GetRegistry(r.Context())).Counter("timeout_requests_total").Inc()

					detail := problem.NewDetail(c.StatusCode, c.Message)
					_ = detail.Render(w) // Best effort - client may have disconnected
					tw.err = ErrTimeoutWrite
				}
			}
		})
	}
}

type timeoutWriter struct {
	w    http.ResponseWriter
	h    http.Header
	wbuf bytes.Buffer
	req  *http.Request

	mu          sync.Mutex
	err         error
	wroteHeader bool
	code        int
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.h
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.err != nil {
		return 0, tw.err
	}

	if !tw.wroteHeader {
		tw.writeHeaderLocked(http.StatusOK)
	}

	return tw.wbuf.Write(p)
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.writeHeaderLocked(code)
}

func (tw *timeoutWriter) writeHeaderLocked(code int) {
	if tw.err != nil {
		return
	}

	if tw.wroteHeader {
		return
	}

	tw.wroteHeader = true
	tw.code = code
}

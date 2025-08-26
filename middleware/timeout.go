package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/alexferl/zerohttp/config"
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
func Timeout(opts ...config.TimeoutOption) func(http.Handler) http.Handler {
	cfg := config.DefaultTimeoutConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = config.DefaultTimeoutConfig.Timeout
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = config.DefaultTimeoutConfig.StatusCode
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultTimeoutConfig.ExemptPaths
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			ctx, cancel := context.WithTimeout(r.Context(), cfg.Timeout)
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
				if _, err := w.Write(tw.wbuf.Bytes()); err != nil {
					panic(fmt.Errorf("response write failed: %w", err))
				}
			case <-ctx.Done():
				tw.mu.Lock()
				defer tw.mu.Unlock()

				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					w.WriteHeader(cfg.StatusCode)
					if cfg.Message != "" {
						if _, err := w.Write([]byte(cfg.Message)); err != nil {
							panic(fmt.Errorf("timeout message write failed: %w", err))
						}
					}
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

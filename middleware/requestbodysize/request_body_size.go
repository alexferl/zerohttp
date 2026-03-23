package requestbodysize

import (
	"net/http"

	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/metrics"
)

// New creates a request size limiting middleware with the provided configuration
func New(cfg ...Config) func(http.Handler) http.Handler {
	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	// Validate MaxBytes - use default if invalid
	if c.MaxBytes <= 0 {
		c.MaxBytes = DefaultConfig.MaxBytes
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "RequestBodySize")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			// Wrap the response writer to detect 413 status
			lrw := &limitResponseWriter{
				ResponseWriter: w,
				reg:            reg,
			}

			r.Body = http.MaxBytesReader(lrw, r.Body, c.MaxBytes)
			next.ServeHTTP(lrw, r)
		})
	}
}

// limitResponseWriter wraps ResponseWriter to detect when MaxBytesReader triggers a 413
type limitResponseWriter struct {
	http.ResponseWriter
	reg   metrics.Registry
	wrote bool
}

func (lrw *limitResponseWriter) WriteHeader(code int) {
	if !lrw.wrote {
		lrw.wrote = true
		if code == http.StatusRequestEntityTooLarge {
			lrw.reg.Counter("request_body_size_rejected_total").Inc()
		}
	}
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *limitResponseWriter) Write(p []byte) (int, error) {
	if !lrw.wrote {
		lrw.WriteHeader(http.StatusOK)
	}
	return lrw.ResponseWriter.Write(p)
}

// Flush implements http.Flusher to support streaming responses like SSE.
// It passes the flush through to the underlying ResponseWriter if it supports Flusher.
func (lrw *limitResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

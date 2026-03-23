package tracer

import (
	"net/http"

	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/mwutil"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/trace"
)

// New creates a tracer middleware with the provided configuration that traces HTTP requests
func New(tracer trace.Tracer, cfg ...Config) func(http.Handler) http.Handler {
	if tracer == nil {
		tracer = trace.NewNoopTracer()
	}

	c := DefaultConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	mwutil.ValidatePathConfig(c.ExcludedPaths, c.IncludedPaths, "Tracer")

	wrapper := c.Wrap()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mwutil.ShouldProcessMiddleware(r.URL.Path, c.IncludedPaths, c.ExcludedPaths) {
				next.ServeHTTP(w, r)
				return
			}

			spanName := wrapper.GetSpanName(r)
			ctx, span := tracer.Start(r.Context(), spanName,
				trace.WithAttributes(
					trace.String("http.method", r.Method),
					trace.String("http.target", r.URL.Path),
					trace.String("http.scheme", scheme(r)),
					trace.String("http.host", r.Host),
				),
			)
			defer span.End()

			if r.ContentLength > 0 {
				span.SetAttributes(trace.Int64("http.request_content_length", r.ContentLength))
			}

			rw := rwutil.NewResponseWriter(w)
			wrapped := &tracingResponseWriter{
				ResponseWriter: rw,
				span:           span,
			}

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			if wrapped.StatusCode() >= 500 {
				span.SetStatus(trace.CodeError, http.StatusText(wrapped.StatusCode()))
			} else {
				span.SetStatus(trace.CodeOk, "")
			}
		})
	}
}

type tracingResponseWriter struct {
	*rwutil.ResponseWriter
	span trace.Span
}

func (w *tracingResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	w.span.SetAttributes(trace.Int("http.status_code", code))
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get(httpx.HeaderXForwardedProto); scheme != "" {
		return scheme
	}
	return "http"
}

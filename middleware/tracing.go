package middleware

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/trace"
)

// Tracing creates a middleware that traces HTTP requests.
func Tracing(tracer trace.Tracer, cfg ...config.TracerConfig) func(http.Handler) http.Handler {
	if tracer == nil {
		tracer = trace.NewNoopTracer()
	}

	c := config.DefaultTracerConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	wrapper := c.Wrap()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if wrapper.IsExempt(r.URL.Path) {
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

			wrapped := &tracingResponseWriter{
				ResponseWriter: w,
				span:           span,
			}

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			if wrapped.statusCode >= 500 {
				span.SetStatus(trace.CodeError, http.StatusText(wrapped.statusCode))
			} else {
				span.SetStatus(trace.CodeOk, "")
			}
		})
	}
}

type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	span       trace.Span
	written    bool
}

func (w *tracingResponseWriter) WriteHeader(code int) {
	if w.written {
		return
	}
	w.written = true
	w.statusCode = code
	w.span.SetAttributes(trace.Int("http.status_code", code))
	w.ResponseWriter.WriteHeader(code)
}

func (w *tracingResponseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

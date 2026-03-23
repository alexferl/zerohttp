// Package trace provides interfaces for distributed tracing.
//
// This package defines a pluggable tracing interface that allows users to integrate
// their preferred tracing implementation (e.g., [OpenTelemetry], [Jaeger], [Zipkin]) without
// forcing dependencies on the core framework.
//
// # Quick Start
//
// No tracing is enabled by default. To enable tracing:
//
//	app := zh.New(zh.Config{
//	    Tracer: myTracer,
//	})
//	app.Use(tracer.New(myTracer))
//
// # Accessing Spans in Handlers
//
// Once tracing is enabled, access the current span from request context:
//
//	app.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) error {
//	    span := trace.SpanFromContext(r.Context())
//	    span.SetAttributes(
//	        trace.String("user.id", userID),
//	        trace.String("db.operation", "select"),
//	    )
//
//	    // Record errors
//	    if err != nil {
//	        span.RecordError(err)
//	        span.SetStatus(trace.CodeError, "database error")
//	    }
//
//	    return zh.Render.JSON(w, http.StatusOK, user)
//	})
//
// # Creating a Custom Tracer
//
// Implement the [Tracer] and [Span] interfaces:
//
//	type MyTracer struct{}
//
//	func (t *MyTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
//	    span := &MySpan{name: name}
//	    return trace.ContextWithSpan(ctx, span), span
//	}
//
//	type MySpan struct{ name string }
//
//	func (s *MySpan) End() {}
//	func (s *MySpan) SetStatus(code trace.Code, description string) {}
//	func (s *MySpan) SetAttributes(attrs ...trace.Attribute) {}
//	func (s *MySpan) RecordError(err error, opts ...trace.ErrorOption) {}
//
// # Using OpenTelemetry
//
// Wrap an OpenTelemetry tracer to implement the interface:
//
//	type OTelTracer struct {
//	    tracer oteltrace.Tracer
//	}
//
//	func (t *OTelTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
//	    ctx, otelSpan := t.tracer.Start(ctx, name)
//	    return ctx, &OTelSpan{span: otelSpan}
//	}
//
// See examples/middleware/tracing_otel for a complete working example.
//
// [OpenTelemetry]: https://opentelemetry.io/
// [Jaeger]: https://www.jaegertracing.io/
// [Zipkin]: https://zipkin.io/
package trace

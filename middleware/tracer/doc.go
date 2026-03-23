// Package tracer provides distributed tracing middleware.
//
// Extracts and propagates trace context from incoming requests
// and creates spans for request processing.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/tracer"
//
//	// Use with OpenTelemetry
//	tp := sdktrace.NewTracerProvider()
//	otel.SetTracerProvider(tp)
//
//	app.Use(tracer.New(tracer.Config{
//	    Tracer: otel.Tracer("myapp"),
//	}))
//
// # Accessing Spans
//
// Retrieve the current span in handlers:
//
//	span := tracer.GetSpan(r.Context())
//	span.SetAttributes(attribute.String("key", "value"))
package tracer

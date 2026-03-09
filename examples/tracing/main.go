// This example shows how to implement a custom tracer without using any external
// tracing library. This is useful for simple logging or custom trace collection.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	"github.com/alexferl/zerohttp/trace"
)

// SimpleTracer is a custom tracer implementation that logs spans to stdout.
// This is a basic example showing how to implement the trace.Tracer interface.
type SimpleTracer struct{}

func (t *SimpleTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	span := &SimpleSpan{
		name:       name,
		startTime:  time.Now(),
		attributes: make(map[string]any),
	}

	fmt.Printf("[TRACE] Span started: %s\n", name)
	return trace.ContextWithSpan(ctx, span), span
}

// SimpleSpan is a custom span implementation that logs to stdout.
type SimpleSpan struct {
	name       string
	startTime  time.Time
	attributes map[string]any
	statusCode trace.Code
	statusDesc string
}

func (s *SimpleSpan) End() {
	duration := time.Since(s.startTime)
	fmt.Printf("[TRACE] Span ended: %s (duration: %v, status: %d)\n", s.name, duration, s.statusCode)
}

func (s *SimpleSpan) SetStatus(code trace.Code, description string) {
	s.statusCode = code
	s.statusDesc = description
}

func (s *SimpleSpan) SetAttributes(attrs ...trace.Attribute) {
	for _, attr := range attrs {
		s.attributes[attr.Key] = attr.Value
		fmt.Printf("[TRACE]   Attribute: %s = %v\n", attr.Key, attr.Value)
	}
}

func (s *SimpleSpan) RecordError(err error, opts ...trace.ErrorOption) {
	fmt.Printf("[TRACE] Error recorded: %v\n", err)
}

func main() {
	// Create our custom tracer
	tracer := &SimpleTracer{}

	// Configure the server with our custom tracer
	app := zh.New(config.Config{
		Tracer: tracer,
	})

	// Add the tracing middleware
	app.Use(middleware.Tracing(tracer))

	// Regular endpoint
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Hello with custom tracing!"})
	}))

	// Endpoint that simulates an error (to show error status in traces)
	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "Something went wrong"})
	}))

	// Endpoint that accesses the current span
	app.GET("/span-info", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		span := trace.SpanFromContext(r.Context())
		if span != nil {
			// Add custom attribute to the current span
			span.SetAttributes(trace.String("custom.key", "custom-value"))
		}
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Check the trace output for custom attribute"})
	}))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Try these endpoints:")
	fmt.Println("  curl http://localhost:8080/")
	fmt.Println("  curl http://localhost:8080/error")
	fmt.Println("  curl http://localhost:8080/span-info")
	fmt.Println()
	fmt.Println("Watch the console output for trace information!")

	log.Fatal(app.Start())
}

//go:build ignore

// This example shows how to integrate OpenTelemetry with zerohttp.
//
// To run this example, you need to install the OpenTelemetry packages:
//
//	go get go.opentelemetry.io/otel \
//	    go.opentelemetry.io/otel/sdk \
//	    go.opentelemetry.io/otel/exporters/stdout/stdouttrace
//
// This example uses the stdout exporter for demonstration. In production,
// you would typically use a Jaeger, Zipkin, or OTLP exporter.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	"github.com/alexferl/zerohttp/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	oTelTrace "go.opentelemetry.io/otel/trace"
)

// OTelTracer wraps an OpenTelemetry tracer to implement zerohttp's trace.Tracer interface.
type OTelTracer struct {
	tracer oTelTrace.Tracer
}

// NewOTelTracer creates a new OTelTracer wrapper.
func NewOTelTracer(tracer oTelTrace.Tracer) *OTelTracer {
	return &OTelTracer{tracer: tracer}
}

// Start implements trace.Tracer by delegating to the underlying OTel tracer.
func (t *OTelTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	ctx, oTelSpan := t.tracer.Start(ctx, name)
	return ctx, &OTelSpan{span: oTelSpan}
}

// OTelSpan wraps an OpenTelemetry span to implement zerohttp's trace.Span interface.
type OTelSpan struct {
	span oTelTrace.Span
}

func (s *OTelSpan) End() {
	s.span.End()
}

func (s *OTelSpan) SetStatus(code trace.Code, description string) {
	// Map zerohttp codes to OTel codes
	// Note: In OTel, use codes package: codes.Ok, codes.Error, codes.Unset
	s.span.SetStatus(otelCodes(code), description)
}

func otelCodes(code trace.Code) codes.Code {
	switch code {
	case trace.CodeOk:
		return codes.Ok
	case trace.CodeError:
		return codes.Error
	default:
		return codes.Unset
	}
}

func (s *OTelSpan) SetAttributes(attrs ...trace.Attribute) {
	// Note: In a real implementation, you would convert trace.Attribute
	// to otel attribute.KeyValue. This is simplified for the example.
	for _, attr := range attrs {
		// Convert to OTel attribute - simplified
		_ = attr
	}
}

func (s *OTelSpan) RecordError(err error, opts ...trace.ErrorOption) {
	s.span.RecordError(err)
}

func main() {
	// Create stdout exporter for demonstration
	// In production, use jaeger, zipkin, or otlp exporter
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("zerohttp-example"),
		)),
	)
	defer provider.Shutdown(context.Background())

	// Set as global tracer provider
	otel.SetTracerProvider(provider)

	// Create our OTel wrapper tracer
	tracer := NewOTelTracer(provider.Tracer("zerohttp"))

	// Configure the server
	app := zh.New(config.Config{
		Tracer: tracer,
	})

	// Add the tracing middleware
	app.Use(middleware.Tracing(tracer))

	// Regular endpoint
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Hello with OpenTelemetry!"})
	}))

	// Endpoint with custom attributes
	app.GET("/user/:id", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Get user ID from path
		userID := r.PathValue("id")

		// Add custom attribute to span
		span := trace.SpanFromContext(r.Context())
		if span != nil {
			span.SetAttributes(trace.String("user.id", userID))
		}

		return zh.R.JSON(w, http.StatusOK, zh.M{"user_id": userID})
	}))

	// Endpoint that simulates an error
	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("something went wrong")
	}))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Try these endpoints:")
	fmt.Println("  curl http://localhost:8080/")
	fmt.Println("  curl http://localhost:8080/user/123")
	fmt.Println("  curl http://localhost:8080/error")
	fmt.Println()
	fmt.Println("Watch the console output for OpenTelemetry trace information!")

	log.Fatal(app.Start())
}

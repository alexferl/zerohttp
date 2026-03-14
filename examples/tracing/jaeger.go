//go:build ignore

// This example shows how to integrate OpenTelemetry with Jaeger for distributed tracing.
//
// Jaeger accepts OTLP (OpenTelemetry Protocol) directly, so we use the OTLP exporter
// which is the recommended approach (the old Jaeger-specific exporter is deprecated).
//
// To run this example:
//
//  1. Start Jaeger: docker run -d --name jaeger -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one:latest
//  2. Run this example: go run examples/tracing/jaeger.go
//  3. Make some requests: curl http://localhost:8080/ && curl http://localhost:8080/error
//  4. View traces: open http://localhost:16686
//
// Install dependencies:
//
//	go get go.opentelemetry.io/otel \
//	    go.opentelemetry.io/otel/sdk \
//	    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelTrace "go.opentelemetry.io/otel/trace"

	// OTLP exporter - sends traces via OTLP protocol (Jaeger supports this natively)
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OTelTracer wraps an OpenTelemetry tracer to implement zerohttp's trace.Tracer interface.
type OTelTracer struct {
	tracer otelTrace.Tracer
}

// NewOTelTracer creates a new OTelTracer wrapper.
func NewOTelTracer(tracer otelTrace.Tracer) *OTelTracer {
	return &OTelTracer{tracer: tracer}
}

// Start implements trace.Tracer by delegating to the underlying OTel tracer.
func (t *OTelTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	ctx, otelSpan := t.tracer.Start(ctx, name)
	return ctx, &OTelSpan{span: otelSpan}
}

// OTelSpan wraps an OpenTelemetry span to implement zerohttp's trace.Span interface.
type OTelSpan struct {
	span otelTrace.Span
}

func (s *OTelSpan) End() {
	s.span.End()
}

func (s *OTelSpan) SetStatus(code trace.Code, description string) {
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
	// Convert trace.Attribute to otel attribute.KeyValue
	otelAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for _, attr := range attrs {
		kv := attribute.KeyValue{
			Key:   attribute.Key(attr.Key),
			Value: toAttributeValue(attr.Value),
		}
		otelAttrs = append(otelAttrs, kv)
	}
	s.span.SetAttributes(otelAttrs...)
}

// toAttributeValue converts Go types to OTel attribute values
func toAttributeValue(v any) attribute.Value {
	switch val := v.(type) {
	case string:
		return attribute.StringValue(val)
	case int:
		return attribute.IntValue(val)
	case int64:
		return attribute.Int64Value(val)
	case float64:
		return attribute.Float64Value(val)
	case bool:
		return attribute.BoolValue(val)
	default:
		return attribute.StringValue(fmt.Sprint(val))
	}
}

func (s *OTelSpan) RecordError(err error, opts ...trace.ErrorOption) {
	s.span.RecordError(err)
}

func main() {
	// Create OTLP exporter that sends to Jaeger's OTLP endpoint
	// Jaeger listens on port 4318 for OTLP HTTP
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create OTLP exporter: %v", err)
	}

	// Create tracer provider with OTLP exporter
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("zerohttp-jaeger-example"),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	)
	defer func() {
		// Flush and shutdown on exit
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Set as global tracer provider
	otel.SetTracerProvider(provider)

	// Create our OTel wrapper tracer
	tracer := NewOTelTracer(provider.Tracer("zerohttp"))

	// Configure the server
	app := zh.New(config.Config{
		Tracer: config.TracerConfig{
			TracerField: tracer,
		},
	})

	// Add the tracing middleware
	app.Use(middleware.Tracing(tracer))

	// Regular endpoint
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Hello with Jaeger tracing!"})
	}))

	// Endpoint that simulates an error
	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("something went wrong")
	}))

	// Slow endpoint to show timing in traces
	app.GET("/slow", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(100 * time.Millisecond)
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "This was slow"})
	}))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Jaeger UI available at http://localhost:16686")
	fmt.Println()
	fmt.Println("Try these endpoints:")
	fmt.Println("  curl http://localhost:8080/")
	fmt.Println("  curl http://localhost:8080/error")
	fmt.Println("  curl http://localhost:8080/slow")
	fmt.Println()
	fmt.Println("Then view traces at: http://localhost:16686")
	fmt.Println()
	fmt.Println("NOTE: Select 'zerohttp-jaeger-example' from the Service dropdown in Jaeger UI")

	log.Fatal(app.Start())
}

package main

import (
	"context"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	"github.com/alexferl/zerohttp/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// OTelTracer wraps OpenTelemetry tracer for zerohttp
type OTelTracer struct {
	tracer otelTrace.Tracer
}

func NewOTelTracer(tracer otelTrace.Tracer) *OTelTracer {
	return &OTelTracer{tracer: tracer}
}

func (t *OTelTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	ctx, otelSpan := t.tracer.Start(ctx, name)
	return ctx, &OTelSpan{span: otelSpan}
}

type OTelSpan struct {
	span otelTrace.Span
}

func (s *OTelSpan) End()                                             { s.span.End() }
func (s *OTelSpan) SetStatus(code trace.Code, desc string)           { s.span.SetStatus(otelCodes(code), desc) }
func (s *OTelSpan) RecordError(err error, opts ...trace.ErrorOption) { s.span.RecordError(err) }

func (s *OTelSpan) SetAttributes(attrs ...trace.Attribute) {
	for _, attr := range attrs {
		s.span.SetAttributes(toOtelAttribute(attr))
	}
}

func toOtelAttribute(attr trace.Attribute) attribute.KeyValue {
	switch v := attr.Value.(type) {
	case string:
		return attribute.String(attr.Key, v)
	case int:
		return attribute.Int(attr.Key, v)
	case int64:
		return attribute.Int64(attr.Key, v)
	case float64:
		return attribute.Float64(attr.Key, v)
	case bool:
		return attribute.Bool(attr.Key, v)
	default:
		return attribute.String(attr.Key, "")
	}
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

func main() {
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("zerohttp-jaeger-example"),
		)),
	)
	defer provider.Shutdown(context.Background())

	otel.SetTracerProvider(provider)
	tracer := NewOTelTracer(provider.Tracer("zerohttp"))

	app := zh.New(config.Config{Tracer: config.TracerConfig{TracerField: tracer}})
	app.Use(middleware.Tracer(tracer))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{"message": "Hello!"})
	}))

	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusInternalServerError, map[string]string{"error": "Oops"})
	}))

	log.Fatal(app.Start())
}

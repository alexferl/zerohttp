package main

import (
	"context"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	"github.com/alexferl/zerohttp/trace"
)

// SimpleTracer logs spans to stdout
type SimpleTracer struct{}

func (t *SimpleTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	span := &SimpleSpan{name: name, startTime: time.Now()}
	return trace.ContextWithSpan(ctx, span), span
}

type SimpleSpan struct {
	name      string
	startTime time.Time
}

func (s *SimpleSpan) End() {
	log.Printf("[TRACE] %s (duration: %v)", s.name, time.Since(s.startTime))
}

func (s *SimpleSpan) SetStatus(code trace.Code, description string) {}
func (s *SimpleSpan) SetAttributes(attrs ...trace.Attribute)        {}
func (s *SimpleSpan) RecordError(err error, opts ...trace.ErrorOption) {
	log.Printf("[TRACE] Error: %v", err)
}

func main() {
	tracer := &SimpleTracer{}

	app := zh.New(config.Config{
		Tracer: config.TracerConfig{TracerField: tracer},
	})

	app.Use(middleware.Tracing(tracer))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{"message": "Hello!"})
	}))

	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusInternalServerError, map[string]string{"error": "Something went wrong"})
	}))

	log.Fatal(app.Start())
}

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	app := zh.New()

	// The RequestID middleware is enabled by default and adds
	// X-Request-ID header to responses and sets it in the context

	// Endpoint that demonstrates accessing request ID from context
	app.GET("/trace", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Access the request ID from context (added by RequestID middleware)
		requestID := middleware.GetRequestID(r.Context())

		// Simulate some work that uses the request ID for logging/tracing
		start := time.Now()
		doWork(r.Context())
		duration := time.Since(start)

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"request_id":  requestID,
			"duration_ms": duration.Milliseconds(),
			"message":     "Request traced successfully",
		})
	}))

	// Endpoint that simulates distributed tracing across services
	app.GET("/chain", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		requestID := middleware.GetRequestID(r.Context())

		// In a real app, this would call another service with the request ID
		// propagated in the outgoing request headers
		subRequestID := callDownstreamService(r.Context(), requestID)

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"request_id":    requestID,
			"downstream_id": subRequestID,
			"chain":         "complete",
		})
	}))

	log.Fatal(app.Start())
}

// doWork simulates some async work that uses the request context
func doWork(ctx context.Context) {
	requestID := middleware.GetRequestID(ctx)
	_ = requestID // In real code, you'd log this or include in spans

	// Simulate work
	time.Sleep(10 * time.Millisecond)
}

// callDownstreamService simulates calling another service with request ID propagation
func callDownstreamService(_ context.Context, parentID string) string {
	// In real code, you'd:
	// 1. Create a child span or continue the trace
	// 2. Propagate the request ID in outgoing headers (X-Request-ID)
	// 3. Return the response's trace info
	time.Sleep(5 * time.Millisecond)
	return parentID + "-downstream"
}

package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	// RequestID middleware is enabled by default with:
	// - Header name: X-Request-Id
	// - Generator: crypto/rand based (128 bits entropy)
	app := zh.New()

	// This endpoint shows the request ID from context
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Get request ID from context
		requestID := middleware.GetRequestID(r.Context())

		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message":    "Hello!",
			"request_id": requestID,
		})
	}))

	// This endpoint returns the request ID from the response header
	app.GET("/headers", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// The middleware automatically sets the response header
		requestID := w.Header().Get(httpx.HeaderXRequestId)

		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message":    "Check response headers too!",
			"request_id": requestID,
		})
	}))

	// This endpoint uses a custom request ID generator (UUID style)
	app.GET("/custom", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		requestID := middleware.GetRequestID(r.Context())

		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"message":    "Custom generator used",
			"request_id": requestID,
		})
	}),
		middleware.RequestID(config.RequestIDConfig{
			Header: httpx.HeaderXRequestId,
			Generator: func() string {
				// Simple UUID-like format for demo purposes
				// In production, use github.com/google/uuid
				return "custom-" + config.GenerateRequestID()[:16]
			},
		}),
	)

	log.Fatal(app.Start())
}

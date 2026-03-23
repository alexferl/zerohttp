package main

import (
	"log"
	"net/http"
	"sync/atomic"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/circuitbreaker"
)

var requestCount atomic.Int32

func main() {
	app := zh.New()

	// Add circuit breaker middleware with custom config
	app.Use(circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold:    3,
		RecoveryTimeout:     5,
		SuccessThreshold:    2,
		MaxHalfOpenRequests: 1,
		IsFailure: func(r *http.Request, statusCode int) bool {
			return statusCode >= 500
		},
		OpenStatusCode: http.StatusServiceUnavailable,
		OpenMessage:    "Circuit breaker is OPEN - service temporarily unavailable",
	}))

	// Endpoint that fails intermittently (to trigger circuit breaker)
	app.GET("/flaky", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		count := requestCount.Add(1)

		// Fail first 5 requests, then succeed
		if count <= 5 {
			return zh.R.JSON(w, http.StatusInternalServerError, map[string]any{
				"error":   "Service error",
				"request": count,
			})
		}

		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"message": "Service is working",
			"request": count,
		})
	}))

	// Always working endpoint
	app.GET("/healthy", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"status": "healthy",
		})
	}))

	log.Fatal(app.Start())
}

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/middleware/circuitbreaker"
)

// numericIDPattern matches numeric IDs in paths for normalization
var numericIDPattern = regexp.MustCompile(`^/users/\d+`)

func main() {
	// Create server with metrics enabled (default)
	// By default, metrics are served on a separate localhost-bound port (localhost:9090)
	// for security. This prevents exposing metrics to the public internet.
	app := zh.New(zh.Config{
		Metrics: metrics.Config{
			Enabled:       config.Bool(true),
			Endpoint:      "/metrics",
			ExcludedPaths: []string{"/health", "/metrics"},
			PathLabelFunc: func(p string) string {
				// Normalize dynamic paths - replace numeric IDs with placeholders
				// e.g., /users/123 -> /users/{id}
				if numericIDPattern.MatchString(p) {
					return "/users/{id}"
				}
				return p
			},
		},
	})

	// Health check (excluded from metrics)
	app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.Render.Text(w, http.StatusOK, "OK")
	}))

	// Simple endpoint
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.Render.Text(w, http.StatusOK, "Hello, World!")
	}))

	// Simulated slow endpoint
	app.GET("/slow", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		time.Sleep(100 * time.Millisecond)
		return zh.Render.Text(w, http.StatusOK, "Slow response")
	}))

	// Endpoint with custom metrics
	app.GET("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Record custom metric
		if reg := metrics.GetRegistry(r.Context()); reg != nil {
			counter := reg.Counter("user_api_requests_total", "endpoint")
			counter.WithLabelValues("list_users").Inc()
		}

		return zh.Render.JSON(w, http.StatusOK, []string{"alice", "bob", "charlie"})
	}))

	// User detail endpoint - demonstrates path normalization in metrics
	// Both /users/123 and /users/456 will be recorded as /users/{id}
	app.GET("/users/:id", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		return zh.Render.JSON(w, http.StatusOK, zh.M{"id": id, "name": "user-" + id})
	}))

	// Business metrics example - order processing
	app.POST("/api/orders", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Access the metrics registry
		if reg := metrics.GetRegistry(r.Context()); reg != nil {
			// Order value histogram
			orderValue := reg.Histogram("order_value_usd",
				[]float64{10, 50, 100, 500, 1000, 5000},
				"region")

			// Orders counter
			ordersCounter := reg.Counter("orders_processed_total", "status", "region")

			region := r.Header.Get("X-Region")
			if region == "" {
				region = "unknown"
			}

			// Simulate processing an order
			orderAmount := 150.00 // In real app, this comes from request body

			// Record metrics
			orderValue.WithLabelValues(region).Observe(orderAmount)
			ordersCounter.WithLabelValues("success", region).Inc()

			fmt.Printf("Order processed: amount=%.2f region=%s\n", orderAmount, region)
		}

		return zh.Render.JSON(w, http.StatusCreated, zh.M{"status": "created"})
	}))

	// Error endpoint - demonstrates error response metrics
	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.Render.JSON(w, http.StatusInternalServerError, zh.M{"error": "intentional error"})
	}))

	// Panic endpoint - demonstrates recovery middleware and panic metrics
	app.GET("/panic", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		panic("intentional panic for demonstration")
	}))

	// Circuit breaker example - demonstrates circuit breaker metrics collection
	// Records: circuit_breaker_state, circuit_breaker_requests_total, circuit_breaker_failures_total, circuit_breaker_trips_total
	app.Group(func(flaky zh.Router) {
		flaky.Use(circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 3,
			RecoveryTimeout:  5 * time.Second,
			SuccessThreshold: 2,
			OpenStatusCode:   http.StatusServiceUnavailable,
			OpenMessage:      "Service temporarily unavailable (circuit open)",
		}))
		flaky.GET("/flaky", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			// Simulate flaky service - 50% failure rate
			if rand.Float32() < 0.5 {
				return zh.Render.JSON(w, http.StatusInternalServerError, zh.M{"error": "service error"})
			}
			return zh.Render.JSON(w, http.StatusOK, zh.M{"data": "success", "timestamp": time.Now().Unix()})
		}))
	})

	log.Fatal(app.Start())
}

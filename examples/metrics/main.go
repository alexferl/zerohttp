package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	// Create server with metrics enabled (default)
	// By default, metrics are served on a separate localhost-bound port (localhost:9090)
	// for security. This prevents exposing metrics to the public internet.
	app := zh.New(config.Config{
		Metrics: config.MetricsConfig{
			Enabled:      true,
			Endpoint:     "/metrics",
			ServerAddr:   "localhost:9090", // Separate metrics server on localhost
			ExcludePaths: []string{"/health", "/metrics"},
			PathLabelFunc: func(p string) string {
				// Normalize dynamic paths - replace IDs with placeholders
				// e.g., /users/123 -> /users/{id}
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

	// Panic endpoint - demonstrates recovery middleware and panic metrics
	app.GET("/panic", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		panic("intentional panic for demonstration")
	}))

	// Circuit breaker example - demonstrates circuit breaker metrics collection
	// Records: circuit_breaker_state, circuit_breaker_requests_total, circuit_breaker_failures_total, circuit_breaker_trips_total
	app.Group(func(flaky zh.Router) {
		flaky.Use(middleware.CircuitBreaker(config.CircuitBreakerConfig{
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

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Metrics available at http://localhost:9090/metrics (localhost only)")
	fmt.Println("")
	fmt.Println("Try these commands:")
	fmt.Println("  curl http://localhost:8080/")
	fmt.Println("  curl http://localhost:8080/slow")
	fmt.Println("  curl http://localhost:8080/api/users")
	fmt.Println("  curl -X POST http://localhost:8080/api/orders -H 'X-Region: us-east'")
	fmt.Println("  curl http://localhost:8080/panic")
	fmt.Println("  curl http://localhost:8080/flaky/data  (50% failure rate, circuit breaker)")
	fmt.Println("  curl http://localhost:9090/metrics")

	log.Fatal(app.Start())
}

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
)

func main() {
	// Create server with metrics enabled (default)
	app := zerohttp.New(config.Config{
		Addr: ":8080",
		Metrics: config.MetricsConfig{
			Enabled:      true,
			Endpoint:     "/metrics",
			ExcludePaths: []string{"/health"}, // Don't track health checks
			PathLabelFunc: func(p string) string {
				// Normalize dynamic paths - replace IDs with placeholders
				// e.g., /users/123 -> /users/{id}
				return p
			},
		},
	})

	// Health check (excluded from metrics)
	app.GET("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Simple endpoint
	app.GET("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, World!"))
	}))

	// Simulated slow endpoint
	app.GET("/slow", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("Slow response"))
	}))

	// Endpoint with custom metrics
	app.GET("/api/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record custom metric
		if reg := metrics.GetRegistry(r.Context()); reg != nil {
			counter := reg.Counter("user_api_requests_total", "endpoint")
			counter.WithLabelValues("list_users").Inc()
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["alice", "bob", "charlie"]`))
	}))

	// Business metrics example - order processing
	app.POST("/api/orders", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status": "created"}`))
	}))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Metrics available at http://localhost:8080/metrics")
	fmt.Println("")
	fmt.Println("Try these commands:")
	fmt.Println("  curl http://localhost:8080/")
	fmt.Println("  curl http://localhost:8080/slow")
	fmt.Println("  curl http://localhost:8080/api/users")
	fmt.Println("  curl -X POST http://localhost:8080/api/orders -H 'X-Region: us-east'")
	fmt.Println("  curl http://localhost:8080/metrics")

	if err := app.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

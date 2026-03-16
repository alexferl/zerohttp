// Package metrics provides [Prometheus]-compatible metrics collection with zero external dependencies.
//
// The metrics system provides counters, gauges, and histograms that can be used
// throughout your application. Metrics are automatically collected for HTTP
// requests when enabled.
//
// # Quick Start
//
// Metrics are enabled by default and exposed at /metrics:
//
//	app := zh.New() // Metrics automatically available at /metrics
//
//	// Access metrics in handlers
//	app.GET("/orders", func(w http.ResponseWriter, r *http.Request) error {
//	    reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))
//	    counter := reg.Counter("orders_total", "status")
//	    counter.WithLabelValues("completed").Inc()
//	    return zh.Render.JSON(w, http.StatusOK, order)
//	})
//
// # Metric Types
//
// Counter - Monotonically increasing values:
//
//	counter := reg.Counter("requests_total", "endpoint", "status")
//	counter.WithLabelValues("/api/users", "200").Inc()
//
// Gauge - Values that can go up or down:
//
//	gauge := reg.Gauge("active_connections", "service")
//	gauge.WithLabelValues("api").Set(42)
//
// Histogram - Sample observations into buckets:
//
//	hist := reg.Histogram("request_duration_seconds",
//	    []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
//	    "endpoint")
//	hist.WithLabelValues("/api/users").Observe(0.045)
//
// # Configuration
//
// Customize metrics behavior:
//
//	app := zh.New(config.Config{
//	    Metrics: config.MetricsConfig{
//	        Enabled:      true,
//	        Endpoint:     "/metrics",
//	        ExcludePaths: []string{"/health", "/readyz"},
//	        CustomLabels: func(r *http.Request) map[string]string {
//	            return map[string]string{
//	                "tenant": r.Header.Get("X-Tenant-ID"),
//	            }
//	        },
//	    },
//	})
//
// # Safe Registry
//
// Use [SafeRegistry] to handle cases where metrics might be disabled:
//
//	reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))
//	// Works even if metrics are disabled - becomes a no-op
//
// # Built-in HTTP Metrics
//
// The following metrics are collected automatically:
//   - http_requests_total - Total requests by method, status, path
//   - http_request_duration_seconds - Request latency distribution
//   - http_request_size_bytes - Request body size distribution
//   - http_response_size_bytes - Response body size distribution
//   - http_requests_in_flight - Currently processing requests
//
// [Prometheus]: https://prometheus.io/
package metrics

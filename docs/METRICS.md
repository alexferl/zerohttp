# Metrics

zerohttp includes built-in Prometheus-compatible metrics collection with zero external dependencies.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Available Metrics](#available-metrics)
- [Custom Labels](#custom-labels)
- [Custom Metrics](#custom-metrics)
- [Middleware Integration](#middleware-integration)
- [Excluding Paths](#excluding-paths)

## Overview

The metrics system provides:

- **Zero dependencies**: Implements Prometheus text format without external libraries
- **Built-in HTTP metrics**: Request counts, durations, sizes, and in-flight requests
- **Custom metrics**: Create your own counters, gauges, and histograms
- **Label support**: Dynamic labels for high-cardinality data
- **Prometheus compatible**: Works with Prometheus, Grafana, Datadog, and other tools

## Quick Start

Metrics are enabled by default and exposed at `/metrics`:

```go
package main

import (
    "log"

    zh "github.com/alexferl/zerohttp"
)

func main() {
    app := zh.New()

    // Metrics are automatically available at /metrics
    // No additional setup required

    app.GET("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        return zh.Render.JSON(w, http.StatusOK, zh.M{"users": []string{"alice", "bob"}})
    }))

    log.Fatal(app.Start())
}
```

View metrics:

```bash
curl http://localhost:8080/metrics
```

## Configuration

Customize metrics behavior via `config.MetricsConfig`:

```go
app := zh.New(config.Config{
    Metrics: config.MetricsConfig{
        // Enable/disable metrics (default: true)
        Enabled: true,

        // Metrics endpoint path (default: "/metrics")
        Endpoint: "/metrics",

        // Histogram buckets for request duration in seconds
        // Default: {0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
        DurationBuckets: []float64{0.001, 0.01, 0.1, 1, 10},

        // Histogram buckets for request/response size in bytes
        // Default: {100, 1000, 10000, 100000, 1000000, 10000000}
        SizeBuckets: []float64{1000, 10000, 100000},

        // Paths to exclude from metrics (e.g., health checks)
        ExcludePaths: []string{"/health", "/readyz", "/metrics"},

        // Transform path for labeling (e.g., normalize IDs)
        PathLabelFunc: func(path string) string {
            // Example: /users/123 -> /users/{id}
            if strings.HasPrefix(path, "/users/") {
                return "/users/{id}"
            }
            return path
        },

        // Add custom labels to all metrics
        CustomLabels: func(r *http.Request) map[string]string {
            return map[string]string{
                "tenant": r.Header.Get("X-Tenant-ID"),
                "region": "us-east",
            }
        },
    },
})
```

## Available Metrics

The following metrics are automatically collected:

| Metric                          | Type      | Description                     | Labels               |
|---------------------------------|-----------|---------------------------------|----------------------|
| `http_requests_total`           | Counter   | Total HTTP requests             | method, status, path |
| `http_request_duration_seconds` | Histogram | Request latency distribution    | method, status, path |
| `http_request_size_bytes`       | Histogram | Request body size distribution  | method, path         |
| `http_response_size_bytes`      | Histogram | Response body size distribution | method, status, path |
| `http_requests_in_flight`       | Gauge     | Currently processing requests   | method, path         |

### Middleware Metrics

Several built-in middlewares also record their own metrics:

| Middleware      | Metric                             | Type      | Description                        | Labels         |
|-----------------|------------------------------------|-----------|------------------------------------|----------------|
| BasicAuth       | `basic_auth_requests_total`        | Counter   | Authentication attempts            | result         |
| CircuitBreaker  | `circuit_breaker_state`            | Gauge     | Current circuit state              | key            |
|                 | `circuit_breaker_requests_total`   | Counter   | Requests through circuit breaker   | key, result    |
|                 | `circuit_breaker_failures_total`   | Counter   | Circuit breaker failures           | key            |
|                 | `circuit_breaker_trips_total`      | Counter   | Circuit breaker trips              | key            |
| Compress        | `compress_requests_total`          | Counter   | Compression requests by encoding   | encoding       |
| CORS            | `cors_preflight_requests_total`    | Counter   | Preflight requests handled         | -              |
|                 | `cors_requests_total`              | Counter   | CORS requests by result            | origin         |
| CSRF            | `csrf_rejected_total`              | Counter   | CSRF rejections                    | reason         |
| ETag            | `etag_requests_total`              | Counter   | ETag cache hits/misses             | result         |
|                 | `etag_generated_total`             | Counter   | Total ETags generated              | -              |
| JWTAuth         | `jwt_auth_requests_total`          | Counter   | JWT authentication attempts        | result         |
| HMACAuth        | `hmac_auth_requests_total`         | Counter   | HMAC authentication attempts       | result         |
| RateLimit       | `ratelimit_allowed_total`          | Counter   | Allowed requests                   | key            |
|                 | `ratelimit_rejected_total`         | Counter   | Rejected requests                  | key            |
|                 | `ratelimit_remaining`              | Gauge     | Remaining requests in window       | key            |
| Recover         | `recover_panics_total`             | Counter   | Recovered panics                   | -              |
| RequestBodySize | `request_body_size_rejected_total` | Counter   | Requests rejected (too large)      | -              |
| ReverseProxy    | `proxy_requests_total`             | Counter   | Proxy requests                     | target, status |
|                 | `proxy_request_duration_seconds`   | Histogram | Proxy latency                      | target         |
| Timeout         | `timeout_requests_total`           | Counter   | Timed out requests                 | -              |

### Example Output

```
# HELP http_requests_total Total http_requests_total
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200",path="/api/users"} 1024
http_requests_total{method="POST",status="201",path="/api/users"} 56

# HELP http_request_duration_seconds Distribution of http_request_duration_seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",status="200",path="/api/users",le="0.1"} 900
http_request_duration_seconds_bucket{method="GET",status="200",path="/api/users",le="0.5"} 1020
http_request_duration_seconds_bucket{method="GET",status="200",path="/api/users",le="+Inf"} 1024
http_request_duration_seconds_sum{method="GET",status="200",path="/api/users"} 45.2
http_request_duration_seconds_count{method="GET",status="200",path="/api/users"} 1024

# HELP http_requests_in_flight Current http_requests_in_flight
# TYPE http_requests_in_flight gauge
http_requests_in_flight{method="GET",path="/api/users"} 3
```

## Custom Labels

Add dynamic labels to all HTTP metrics using `CustomLabels`:

```go
app := zh.New(config.Config{
    Metrics: config.MetricsConfig{
        CustomLabels: func(r *http.Request) map[string]string {
            return map[string]string{
                "tenant":   r.Header.Get("X-Tenant-ID"),
                "region":   os.Getenv("REGION"),
                "version":  "v1.2.3",
            }
        },
    },
})
```

Label keys are extracted from the first request and remain consistent for all subsequent requests. Missing labels default to empty strings.

## Custom Metrics

Create your own metrics using the registry from request context:

```go
import "github.com/alexferl/zerohttp/metrics"

app.POST("/orders", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Process order...

    // Get registry from context
    reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

    // Counter with labels
    orderCounter := reg.Counter("orders_processed_total", "status", "region")
    orderCounter.WithLabelValues("success", "us-east").Inc()

    // Histogram for order values
    orderValue := reg.Histogram("order_value_usd",
        []float64{10, 50, 100, 500, 1000, 5000},
        "region")
    orderValue.WithLabelValues("us-east").Observe(150.0)

    // Gauge for current queue depth
    queueDepth := reg.Gauge("order_queue_depth", "priority")
    queueDepth.WithLabelValues("high").Set(42)

    w.WriteHeader(http.StatusCreated)
}))
```

### Metric Types

**Counter** - Monotonically increasing values:
```go
counter := reg.Counter("requests_total", "endpoint")
counter.WithLabelValues("create_user").Inc()
counter.WithLabelValues("create_user").Add(5)
```

**Gauge** - Values that can go up or down:
```go
gauge := reg.Gauge("active_connections", "service")
gauge.WithLabelValues("api").Set(100)
gauge.WithLabelValues("api").Inc()
gauge.WithLabelValues("api").Dec()
gauge.WithLabelValues("api").Add(10)
gauge.WithLabelValues("api").Sub(5)
```

**Histogram** - Sample observations into buckets:
```go
hist := reg.Histogram("request_latency_seconds",
    []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
    "endpoint")
hist.WithLabelValues("/api/users").Observe(0.045)
```

## Middleware Integration

Other middlewares can access the registry to record their own metrics:

```go
func MyMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get safe registry - works even if metrics disabled
            reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

            // Record custom metric
            counter := reg.Counter("my_middleware_requests_total", "result")

            start := time.Now()
            next.ServeHTTP(w, r)
            duration := time.Since(start)

            if duration > 100*time.Millisecond {
                counter.WithLabelValues("slow").Inc()
            } else {
                counter.WithLabelValues("fast").Inc()
            }
        })
    }
}
```

## Excluding Paths

Exclude specific paths from metrics collection:

```go
app := zh.New(config.Config{
    Metrics: config.MetricsConfig{
        // Exclude health checks and metrics endpoint
        ExcludePaths: []string{
            "/health",
            "/readyz",
            "/livez",
            "/metrics",
            "/debug/pprof",  // pprof endpoints
        },
    },
})
```

Excluded paths are still served but don't generate metrics, reducing noise from health check polling.

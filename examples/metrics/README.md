# Metrics Example

This example demonstrates the built-in Prometheus-compatible metrics middleware.

## Features

- **Automatic HTTP metrics**: request count, duration, request/response size
- **Custom metrics**: create counters, gauges, and histograms in your handlers
- **Context-based registry access**: middlewares retrieve registry from request context
- **Prometheus format**: metrics exposed in standard text format at `/metrics`

## Running

```bash
go run main.go
```

## Testing

Make some requests:

```bash
# Basic requests
curl http://localhost:8080/
curl http://localhost:8080/slow
curl http://localhost:8080/api/users

# POST with region header
curl -X POST http://localhost:8080/api/orders \
  -H 'Content-Type: application/json' \
  -H 'X-Region: us-east' \
  -d '{"item": "widget", "amount": 150.00}'

# View metrics
curl http://localhost:8080/metrics
```

## Sample Output

```
# HELP http_requests_total Total http_requests_total
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 5
http_requests_total{method="GET",path="/api/users",status="200"} 3
http_requests_total{method="POST",path="/api/orders",status="201"} 2

# HELP http_request_duration_seconds Distribution of http_request_duration_seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/",status="200",le="0.005"} 4
http_request_duration_seconds_bucket{method="GET",path="/",status="200",le="0.01"} 5
...
http_request_duration_seconds_sum{method="GET",path="/",status="200"} 0.012
http_request_duration_seconds_count{method="GET",path="/",status="200"} 5

# HELP orders_processed_total Total orders_processed_total
# TYPE orders_processed_total counter
orders_processed_total{region="us-east",status="success"} 2

# HELP order_value_usd Distribution of order_value_usd
# TYPE order_value_usd histogram
order_value_usd_bucket{region="us-east",le="100"} 0
order_value_usd_bucket{region="us-east",le="500"} 2
...
```

## Configuration

```go
app := zerohttp.New(config.Config{
    Metrics: config.MetricsConfig{
        Enabled:      true,                    // Enable metrics (default)
        Endpoint:     "/metrics",              // Metrics endpoint path
        ExcludePaths: []string{"/health"},     // Paths to exclude
        DurationBuckets: []float64{            // Histogram buckets (seconds)
            0.005, 0.01, 0.025, 0.05, 0.1,
            0.25, 0.5, 1, 2.5, 5, 10,
        },
        SizeBuckets: []float64{               // Histogram buckets (bytes)
            100, 1000, 10000, 100000,
            1000000, 10000000,
        },
        PathLabelFunc: func(p string) string { // Normalize paths
            return p
        },
    },
})
```

## Custom Metrics

```go
app.GET("/api/orders", func(w http.ResponseWriter, r *http.Request) {
    // Get registry from context
    if reg := zerohttp.GetMetricsRegistry(r.Context()); reg != nil {
        // Create counter
        counter := reg.Counter("orders_total", "status")
        counter.WithLabelValues("success").Inc()

        // Create histogram
        hist := reg.Histogram("order_value",
            []float64{10, 50, 100, 500}, "currency")
        hist.WithLabelValues("usd").Observe(150.00)

        // Create gauge
        gauge := reg.Gauge("active_orders", "region")
        gauge.WithLabelValues("us-east").Inc()
    }
})
```

## Disable Metrics

```go
app := zerohttp.New(config.Config{
    Metrics: config.MetricsConfig{
        Enabled: false,
    },
})
```

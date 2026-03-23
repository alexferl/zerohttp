# Metrics Example

This example demonstrates Prometheus-style metrics collection with automatic HTTP metrics and custom business metrics.

## Features

- Automatic HTTP request metrics (duration, count, size)
- Separate metrics server on localhost for security
- Custom business metrics (counters, histograms)
- Path normalization for dynamic routes
- Circuit breaker metrics
- Panic/recovery metrics

## Default Metrics

When metrics are enabled, the following are automatically collected:

| Metric                           | Type      | Description                             |
|----------------------------------|-----------|-----------------------------------------|
| `http_requests_total`            | Counter   | Total requests by method, path, status  |
| `http_request_duration_seconds`  | Histogram | Request latency by method, path, status |
| `http_request_size_bytes`        | Histogram | Request body size                       |
| `http_response_size_bytes`       | Histogram | Response body size                      |
| `http_panics_total`              | Counter   | Total recovered panics                  |
| `circuit_breaker_state`          | Gauge     | Current circuit breaker state           |
| `circuit_breaker_requests_total` | Counter   | Total requests through circuit breaker  |
| `circuit_breaker_failures_total` | Counter   | Total failures through circuit breaker  |
| `circuit_breaker_trips_total`    | Counter   | Total circuit breaker trips             |

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.
Metrics are available at `http://localhost:9090/metrics` (localhost only).

## Endpoints

| Method | Endpoint      | Description                           |
|--------|---------------|---------------------------------------|
| `GET`  | `/`           | Hello world                           |
| `GET`  | `/health`     | Health check (excluded from metrics)  |
| `GET`  | `/slow`       | Slow endpoint (~100ms)                |
| `GET`  | `/api/users`  | List users with custom counter metric |
| `GET`  | `/users/:id`  | User detail (path normalization demo) |
| `POST` | `/api/orders` | Create order with histogram metric    |
| `GET`  | `/error`      | Returns 500 error                     |
| `GET`  | `/panic`      | Triggers panic (recovered)            |
| `GET`  | `/flaky`      | 50% failure rate (circuit breaker)    |

## Test Commands

### Basic endpoints
```bash
curl http://localhost:8080/
curl http://localhost:8080/health
curl http://localhost:8080/slow
```

### Custom metrics endpoints
```bash
curl http://localhost:8080/api/users
curl http://localhost:8080/users/123
curl http://localhost:8080/users/456
curl -X POST http://localhost:8080/api/orders -H 'X-Region: us-east'
```

### Error and panic endpoints
```bash
curl http://localhost:8080/error
curl http://localhost:8080/panic
```

### Circuit breaker endpoint
```bash
# Run multiple times to see circuit breaker in action
for i in {1..10}; do curl http://localhost:8080/flaky; echo; done
```

### View metrics
```bash
# All metrics in Prometheus format
curl http://localhost:9090/metrics

# Filter for specific metrics
curl http://localhost:9090/metrics | grep http_requests_total
curl http://localhost:9090/metrics | grep user_api_requests_total
curl http://localhost:9090/metrics | grep circuit_breaker
```

## Security Considerations

By default, metrics are served on a separate localhost-bound server (`localhost:9090`). This prevents exposing potentially sensitive metrics (path names, error rates, business data) to the public internet.

Production deployments should:
- Keep metrics on a separate port/interface
- Use firewall rules to restrict metrics access
- Consider adding authentication for metrics endpoints

# Reverse Proxy Example

This example demonstrates zerohttp's reverse proxy middleware for routing requests to upstream servers and load balancing.

## Features

- Simple reverse proxy with path stripping
- Load balancing with health checks
- Round-robin and least-connections algorithms
- Custom headers and path rewriting

## Running the Example

Start a backend server on port 8081:
```bash
go run examples/core/hello_world/main.go
```

Then start the proxy:
```bash
go run .
```

The proxy starts on `http://localhost:8080`.

## Endpoints

| Endpoint       | Description                          |
|----------------|--------------------------------------|
| `GET /`        | Direct response from proxy           |
| `GET /api/*`   | Proxied to localhost:8081            |
| `GET /lb/*`    | Load balanced across backends        |

## Test Commands

### Direct response
```bash
curl http://localhost:8080/
```

### Proxied request (path stripped)
```bash
curl http://localhost:8080/api/users
# Routes to http://localhost:8081/users
```

### Load balanced request
```bash
for i in {1..5}; do curl http://localhost:8080/lb/; done
```

## Configuration

### Simple Proxy
```go
rp, cleanup := middleware.ReverseProxy(config.ReverseProxyConfig{
    Target:      "http://localhost:8081",
    StripPrefix: "/api",
})
defer cleanup()
```

### Load Balancer
```go
rp, cleanup := middleware.ReverseProxy(config.ReverseProxyConfig{
    Targets: []config.Backend{
        {Target: "http://backend1:8081", Weight: 1},
        {Target: "http://backend2:8081", Weight: 2},
    },
    LoadBalancer:        config.RoundRobin,
    HealthCheckInterval: 10 * time.Second,
    HealthCheckPath:     "/health",
})
defer cleanup()
```

### Custom Headers
```go
rp, cleanup := middleware.ReverseProxy(config.ReverseProxyConfig{
    Target: "http://api.example.com",
    SetHeaders: map[string]string{
        "X-Proxy-By": "zerohttp",
    },
    RemoveHeaders: []string{"X-Internal-Token"},
})
defer cleanup()
```

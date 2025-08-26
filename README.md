# zerohttp [![Go Report Card](https://goreportcard.com/badge/github.com/alexferl/zerohttp)](https://goreportcard.com/report/github.com/alexferl/zerohttp) [![codecov](https://codecov.io/gh/alexferl/zerohttp/branch/master/graph/badge.svg)](https://codecov.io/gh/alexferl/zerohttp)

A lightweight HTTP framework for Go built on top of the standard `net/http` library. Designed for simplicity, developer productivity, and security.

## Features

- **Lightweight**: Built on Go's standard `net/http` with minimal overhead
- **Zero Dependencies**: No external dependencies except `golang.org/x/crypto` for AutoTLS
- **Secure by Default**: Automatically applies essential security middlewares out of the box
- **Response Rendering**: Built-in support for JSON, HTML, text, and file responses
- **Request Binding**: JSON request body parsing
- **Problem Details**: RFC 9457 Problem Details for HTTP APIs error responses
- **Flexible Routing**: Method-based routing with route groups and parameter support
- **Middleware Support**: Comprehensive middleware system with built-in security, logging, and utility middlewares
- **Built-in Security**: CORS, rate limiting, request body size limits, security headers, and more
- **Auto-TLS**: Built-in Let's Encrypt support with automatic certificate management
- **Request Tracing**: Built-in request ID generation and propagation
- **Circuit Breaker**: Prevent cascading failures with configurable circuit breaker middleware
- **Structured Logging**: Integrated structured logging with customizable fields
- **Health Checks**: Kubernetes-compatible health check endpoints with customizable handlers


## Requirements

- **Go 1.25 or later**
- **No external dependencies** (except `golang.org/x/crypto` for AutoTLS features)


## Secure by Default

zerohttp applies security best practices automatically with these default middlewares:

- **Request ID**: Generates unique request IDs for tracing and debugging
- **Panic Recovery**: Gracefully handles panics with stack trace logging
- **Request Body Size Limits**: Prevents DoS attacks from large request bodies
- **Security Headers**: Sets essential security headers (CSP, HSTS, X-Frame-Options, etc.)
- **Request Logging**: Comprehensive request/response logging with security context

These middlewares are enabled by default but can be customized or disabled as needed.


## Installation

```bash
go get github.com/alexferl/zerohttp
```


## Quick Start

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"

    zh "github.com/alexferl/zerohttp"
)

func main() {
    app := zh.New()

    // Using standard net/http - full control
    app.GET("/hello-std", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)

        response := map[string]string{"message": "Hello from standard library!"}
        json.NewEncoder(w).Encode(response)
    }))

    // Using zerohttp helpers - more concise
    app.GET("/hello", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        return zh.Render.JSON(w, 200, zh.M{"message": "Hello from zerohttp!"})
    }))

    log.Fatal(app.Start())
}
```


## Response Rendering

Clean, extensible interfaces for all response types:

```go
// JSON responses (most common)
zh.Render.JSON(w, 200, zh.M{"message": "Hello, World!"})

// Text responses
zh.Render.Text(w, 200, "Plain text response")

// HTML responses
zh.Render.HTML(w, 200, "<h1>Welcome</h1>")

// Binary data
zh.Render.Blob(w, 200, "image/png", pngData)

// Streaming responses
zh.Render.Stream(w, 200, "text/plain", reader)

// File serving with proper headers
zh.Render.File(w, r, "path/to/document.pdf")

// RFC 9457 Problem Details
problem := zh.NewProblemDetail(404, "User not found")
problem.Set("user_id", "123")
zh.Render.ProblemDetail(w, problem)
```

**Short alias available**: Use `zh.R` instead of `zh.Render` for brevity.

## Request Binding

Simple JSON request parsing with validation:

```go
app.POST("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var user struct {
        Name  string `json:"name"`
        Email string `json:"email"`
        Age   int    `json:"age"`
    }

    // Bind JSON with unknown field validation
    if err := zh.Bind.JSON(r.Body, &user); err != nil {
        problem := zh.NewProblemDetail(400, "Invalid request body")
        problem.Set("error", err.Error())
        return zh.Render.ProblemDetail(w, problem)
    }

    // Process user...
    return zh.Render.JSON(w, 201, user)
}))
```

**Short alias available**: Use `zh.B` instead of `zh.Bind` for convenience.

The binder uses `json.Decoder` with `DisallowUnknownFields()` for stricter validation.

## Middleware

zerohttp includes a comprehensive set of built-in middlewares:

```go
app := zerohttp.New()

// Add additional middleware (default security middlewares already applied)
app.Use(
    middleware.CORS(
        config.WithCORSAllowedOrigins([]string{"https://example.com"}),
        config.WithCORSAllowCredentials(true),
    ),
    middleware.RateLimit(
        config.WithRateLimitRate(100),
        config.WithRateLimitWindow(time.Minute),
    ),
)

// Route-specific middleware
app.GET("/admin", adminHandler,
    middleware.BasicAuth(
        config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}),
    ),
    middleware.RequestBodySize(
        config.WithRequestBodySizeMaxBytes(1024 * 1024), // 1MB limit
    ),
)
```


## Route Groups

Organize your routes with groups:

```go
app.Group(func(api zerohttp.Router) {
    api.Use(middleware.RequireAuth())

    api.GET("/users", listUsers)
    api.POST("/users", createUser)
    api.PUT("/users/{id}", updateUser)
    api.DELETE("/users/{id}", deleteUser)
})
```


## Error Handling

Built-in support for RFC 9457 Problem Details:

```go
app.GET("/error", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    problem := zerohttp.NewProblemDetail(400, "Invalid request")
    problem.Set("field", "email")
    problem.Set("reason", "Email address is required")
    return zerohttp.R.ProblemDetail(w, problem)
}))
```


### Validation Errors

Built-in support for validation error responses:

```go
// Using default validation errors
errors := []zerohttp.ValidationError{
    {Detail: "must be a valid email", Pointer: "#/email"},
    {Detail: "must be at least 8 characters", Field: "password"},
}
problem := zerohttp.NewValidationProblemDetail("Validation failed", errors)
return zerohttp.R.ProblemDetail(w, problem)

// Using custom error structures
type CustomError struct {
    Code    string `json:"code"`
    Field   string `json:"field"`
    Message string `json:"message"`
}

customErrors := []CustomError{
    {Code: "INVALID_EMAIL", Field: "email", Message: "Email format is invalid"},
}
problem := zerohttp.NewValidationProblemDetail("Validation failed", customErrors)
return zerohttp.R.ProblemDetail(w, problem)
```


## Configuration

Flexible configuration system with functional options:

```go
app := zerohttp.New(
    // Server configuration
    config.WithAddr(":8080"),
    config.WithServer(&http.Server{
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }),
    config.WithTLSAddr(":8443"),
    config.WithTLSServer(&http.Server{
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }),
    config.WithCertFile("cert.pem"),
    config.WithKeyFile("key.pem"),
    config.WithLogger(myCustomLogger),

    // Configure default middlewares using their respective option containers
    config.WithRequestBodySizeOptions(
        config.WithRequestBodySizeMaxBytes(10*1024*1024), // 10MB
    ),
    config.WithRequestIDOptions(
        config.WithRequestIDHeader("X-Request-ID"),
    ),
    config.WithRecoverOptions(
        config.WithRecoverStackSize(8192),
        config.WithRecoverEnableStackTrace(true),
    ),
    config.WithSecurityHeadersOptions(
        config.WithSecurityHeadersCSP("default-src 'self'"),
        config.WithSecurityHeadersXFrameOptions("SAMEORIGIN"),
        config.WithSecurityHeadersHSTS(
            config.WithHSTSMaxAge(31536000), // 1 year
            config.WithHSTSPreload(true),
        ),
    ),
)
```


## Disabling Default Security

If you need to disable default middlewares:

```go
app := zerohttp.New(
    config.WithDisableDefaultMiddlewares(), // Disable all defaults
    // Or provide custom defaults
    config.WithDefaultMiddlewares([]func(http.Handler) http.Handler{
        middleware.RequestID(),
        middleware.CORS(),
    }),
)
```


## Available Middlewares

- **Authentication**: Basic Auth
- **Security**: CORS, Security Headers, Request Body Size Limits
- **Rate Limiting**: Rate Limit with configurable algorithms
- **Content Handling**: Compress, Content Charset, Content Encoding, Content Type
- **Monitoring**: Request Logger, Circuit Breaker, Timeout, Recover
- **Utilities**: Request ID, Real IP, Trailing Slash, Set Header, No Cache

Each middleware uses functional options for configuration:

```go
// CORS middleware
middleware.CORS(
    config.WithCORSAllowedOrigins([]string{"https://example.com"}),
    config.WithCORSAllowCredentials(true),
)

// Rate limiting
middleware.RateLimit(
    config.WithRateLimitRate(50),
    config.WithRateLimitWindow(time.Minute),
    config.WithRateLimitAlgorithm(config.TokenBucket),
)

// Compression
middleware.Compress(
    config.WithCompressLevel(6),
    config.WithCompressMinSize(1024),
)

// Security headers with HSTS options
middleware.SecurityHeaders(
    config.WithSecurityHeadersCSP("default-src 'self'; script-src 'self' 'unsafe-inline'"),
    config.WithSecurityHeadersHSTS(
        config.WithHSTSMaxAge(31536000),
        config.WithHSTSPreload(true),
    ),
)
```


## Extensible Interfaces

Both rendering and binding use interfaces, making them easy to customize:

```go
// Custom renderer
type MyRenderer struct{}

func (r *MyRenderer) JSON(w http.ResponseWriter, code int, data any) error {
    // Custom JSON rendering logic
    w.Header().Set("X-Custom-JSON", "true")
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    return json.NewEncoder(w).Encode(data)
}

// Replace default
zerohttp.Render = &MyRenderer{}

// Custom binder
type MyBinder struct{}

func (b *MyBinder) JSON(r io.Reader, dst any) error {
    // Custom JSON binding logic
    decoder := json.NewDecoder(r)
    decoder.UseNumber() // Use json.Number instead of float64
    return decoder.Decode(dst)
}

// Replace default
zerohttp.Bind = &MyBinder{}
```


## Auto-TLS with Let's Encrypt

```go
app := zerohttp.New(
    config.WithAutocertManager(zerohttp.NewAutocertManager("/tmp/certs", "example.com")),
)

app.StartAutoTLS("example.com", "www.example.com")
```


## Health Checks

Add Kubernetes-compatible health check endpoints with minimal setup:

```go
import (
    "log"
    "net/http"

    zh "github.com/alexferl/zerohttp"
    "github.com/alexferl/zerohttp/healthcheck"
)

func main() {
    app := zh.New()

    // Add default health endpoints: /livez, /readyz, /startupz
    healthcheck.New(app)

    // Or customize endpoints and handlers
    healthcheck.New(app,
        healthcheck.WithLivenessEndpoint("/health/live"),
        healthcheck.WithReadinessEndpoint("/health/ready"),
        healthcheck.WithReadinessHandler(func(w http.ResponseWriter, r *http.Request) error {
            // Check database connections, dependencies, etc.
            if !isAppReady() {
                return zh.R.Text(w, http.StatusServiceUnavailable, "not ready")
            }
            return zh.R.Text(w, http.StatusOK, "ready")
        }),
        healthcheck.WithStartupEndpoint("/health/startup"),
    )

    log.Fatal(app.Start())
}
```

The health check package provides three standard endpoints:

- **`/livez`** - Liveness probe (is the app running?)
- **`/readyz`** - Readiness probe (is the app ready to handle traffic?)
- **`/startupz`** - Startup probe (has the app finished initializing?)


## Circuit Breaker

Prevent cascading failures with configurable circuit breaker middleware:

```go
// Basic circuit breaker - breaks after 5 failures, recovers after 30s
app.Use(middleware.CircuitBreaker())

// Custom configuration
app.Use(middleware.CircuitBreaker(
    config.WithCircuitBreakerFailureThreshold(3),             // Break after 3 failures
    config.WithCircuitBreakerRecoveryTimeout(10*time.Second), // Try recovery after 10s
    config.WithCircuitBreakerOpenStatusCode(503),             // Return 503 when open
))
```

The circuit breaker operates in three states: **Closed** (normal), **Open** (blocked), and **Half-Open** (testing recovery). It prevents cascading failures when downstream services are unavailable.


## Configuration Reference

The functional options pattern provides structured configuration for all aspects of the server:

### Server Configuration

- `config.WithAddr()` - HTTP server address
- `config.WithTLSAddr()` - HTTPS server address
- `config.WithServer()` - HTTP server settings
- `config.WithTLSServer()` - HTTPS server settings
- `config.WithListener()`/`config.WithTLSListener()` - Custom listeners
- `config.WithCertFile()`/`config.WithKeyFile()` - TLS certificate files
- `config.WithAutocertManager()` - Let's Encrypt integration


### Middleware Configuration

- `config.WithDisableDefaultMiddlewares()` - Disable built-in middlewares
- `config.WithDefaultMiddlewares()` - Custom middleware chain
- `config.WithRequestIDOptions()` - Request ID generation settings
- `config.WithRecoverOptions()` - Panic recovery settings
- `config.WithRequestBodySizeOptions()` - Request body size limits
- `config.WithSecurityHeadersOptions()` - Security header configuration options
- `config.WithRequestLoggerOptions()` - Request logging configuration


### Logging

- `config.WithLogger()` - Custom logger instance

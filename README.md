# zerohttp [![Go Report Card](https://goreportcard.com/badge/github.com/alexferl/zerohttp)](https://goreportcard.com/report/github.com/alexferl/zerohttp) [![Coverage Status](https://coveralls.io/repos/github/alexferl/zerohttp/badge.svg?branch=master)](https://coveralls.io/github/alexferl/zerohttp?branch=master)

**⚠️ This is a pre-v1 release - APIs may change as we work toward a stable v1.0.**

A lightweight, secure-by-default HTTP framework for Go. Built on `net/http` with zero external dependencies.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Secure by Default](#secure-by-default)
  - [Security Notice](#security-notice)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Response Rendering](#response-rendering)
- [Request Binding & Parameters](#request-binding--parameters)
- [Validation](#validation)
- [Error Handling](#error-handling)
  - [Validation Errors](#validation-errors)
- [Route Groups](#route-groups)
- [Automatic OPTIONS Responses](#automatic-options-responses)
- [Middleware](#middleware)
  - [Available Middlewares](#available-middlewares)
- [Server Lifecycle Hooks](#server-lifecycle-hooks)
  - [Startup Hooks](#startup-hooks)
  - [Shutdown Hooks](#shutdown-hooks)
  - [Hook Execution Order](#hook-execution-order)
  - [Error Handling](#error-handling)
- [Health Checks](#health-checks)
- [Circuit Breaker](#circuit-breaker)
- [Metrics](#metrics)
- [Distributed Tracing](#distributed-tracing)
- [Static File Serving](#static-file-serving)
  - [Static File Methods](#static-file-methods)
  - [Fallback Behavior](#fallback-behavior)
  - [API Prefix Exclusions](#api-prefix-exclusions)
- [Extensibility](#extensibility)
  - [Extensible Interfaces](#extensible-interfaces)
  - [Pluggable Features](#pluggable-features)
- [Testing Utilities](#testing-utilities)
- [Profiling](#profiling)
- [Configuration](#configuration)
  - [Server Configuration](#server-configuration)
  - [Middleware Configuration](#middleware-configuration)
  - [Disabling Default Security](#disabling-default-security)


## Features

- **Lightweight**: Built on Go's standard `net/http` with minimal overhead
- **Zero Dependencies**: No external dependencies
- **Secure by Default**: Automatically applies essential security middlewares out of the box
- **Response Rendering**: Built-in support for JSON, HTML, text, and file responses
- **Request Binding**: JSON, form, multipart form, and query parameter parsing with struct tag binding
- **Validation**: Built-in struct tag-based validation with 40+ validators
- **Problem Details**: RFC 9457 Problem Details for HTTP APIs error responses
- **Flexible Routing**: Method-based routing with route groups, parameter support, and automatic OPTIONS responses
- **Middleware Support**: Comprehensive middleware system with built-in security, logging, and utility middlewares
- **Built-in Security**: CORS, rate limiting, request body size limits, security headers, and more
- **HTTP/2 & HTTP/3**: Automatic HTTP/2 support for TLS; optional HTTP/3 via pluggable interface
- **Pluggable Architecture**: Extensible interfaces for Auto-TLS, HTTP/3, WebSocket, WebTransport, SSE, and Validator - bring your own implementations
- **Server-Sent Events**: Built-in SSE support with event replay and broadcast hub for real-time server-to-client streaming
- **Request Tracing**: Built-in request ID generation and propagation
- **Distributed Tracing**: Pluggable tracing interface for OpenTelemetry, Jaeger, or custom implementations
- **Circuit Breaker**: Prevent cascading failures with configurable circuit breaker middleware
- **Metrics**: Built-in Prometheus-compatible metrics with zero dependencies
- **Structured Logging**: Integrated structured logging with customizable fields
- **Health Checks**: Kubernetes-compatible health check endpoints with customizable handlers


## Requirements

- **Go 1.25 or later**

## Secure by Default

zerohttp applies security best practices automatically with these default middlewares:

- **Request ID**: Generates unique request IDs for tracing and debugging
- **Panic Recovery**: Gracefully handles panics with stack trace logging
- **Request Body Size Limits**: Prevents DoS attacks from large request bodies
- **Security Headers**: Sets essential security headers (CSP, HSTS, X-Frame-Options, etc.)
- **Request Logging**: Comprehensive request/response logging with security context

These middlewares are enabled by default but can be customized or disabled as needed.

### Security Notice

We do our best to follow security best practices, but this hasn't been formally audited by security experts. Consider doing your own review if security is critical for your use case.

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
        w.Header().Set(consts.HeaderContentType, consts.MIMEApplicationJSON)
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

> **💡 More Examples:** Check out the [`examples/`](examples/) folder for complete working examples including template rendering, static file serving, middleware usage, advanced configurations and more.

## Response Rendering

Clean, extensible interfaces for all response types:

```go
// JSON responses (most common)
zh.Render.JSON(w, http.StatusOK, zh.M{"message": "Hello, World!"})

// Text responses
zh.Render.Text(w, http.StatusOK, "Plain text response")

// HTML responses
zh.Render.HTML(w, http.StatusOK, "<h1>Welcome</h1>")

// Template rendering with parsed templates
tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.html"))
zh.Render.Template(w, http.StatusOK, tmpl, "index.html", zh.M{"title": "Welcome"})

// Binary data
zh.Render.Blob(w, http.StatusOK, "image/png", pngData)

// Streaming responses
zh.Render.Stream(w, http.StatusOK, "text/plain", reader)

// File serving with proper headers
zh.Render.File(w, r, "path/to/document.pdf")

// RFC 9457 Problem Details
problem := zh.NewProblemDetail(http.StatusUnprocessableEntity, "User not found")
problem.Set("user_id", "123")
zh.Render.ProblemDetail(w, problem)
```

**Short alias available**: Use `zh.R` instead of `zh.Render` for brevity.

## Request Binding & Parameters

Parse request bodies (JSON, forms, multipart) and extract path/query parameters. Run `go doc` for details.

## Validation

Built-in struct tag-based validation with 40+ validators:

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=13,max=120"`
}

app.POST("/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req CreateUserRequest
    if err := zh.Bind.JSON(r, &req); err != nil {
        return err // Automatic Problem Details response
    }

    if err := zh.Validate.Struct(&req); err != nil {
        return err // Automatic validation error response
    }

    // Process valid request...
    return zh.Render.JSON(w, http.StatusCreate, req)
}))
```

Run `go doc github.com/alexferl/zerohttp/internal/validator` for full documentation including all validators.

## Error Handling

Built-in support for RFC 9457 Problem Details:

```go
app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    problem := zh.NewProblemDetail(http.StatusUnprocessableEntity, "Invalid request")
    problem.Set("field", "email")
    problem.Set("reason", "Email address is required")
    return zh.Render.ProblemDetail(w, problem)
}))
```

### Validation Errors

Built-in support for validation error responses:

```go
// Using default validation errors
errors := []zh.ValidationError{
    {Detail: "must be a valid email", Pointer: "#/email"},
    {Detail: "must be at least 8 characters", Field: "password"},
}
// Using Render shortcut
return zh.NewValidationProblemDetail("Validation failed", errors).Render(w)

// Using custom error structures
type CustomError struct {
    Code    string `json:"code"`
    Field   string `json:"field"`
    Message string `json:"message"`
}

customErrors := []CustomError{
    {Code: "INVALID_EMAIL", Field: "email", Message: "Email format is invalid"},
}
return zh.NewValidationProblemDetail("Validation failed", customErrors).Render(w)
```

## Route Groups

Organize your routes with groups:

```go
app.Group(func(api zh.Router) {
    api.Use(middleware.RequireAuth())

    api.GET("/users", listUsers)
    api.POST("/users", createUser)
    api.PUT("/users/{id}", updateUser)
    api.DELETE("/users/{id}", deleteUser)
})
```

## Automatic OPTIONS Responses

The router automatically responds to OPTIONS requests with accurate `Allow` headers listing the registered methods for each path:

```
OPTIONS /api/users HTTP/1.1

HTTP/1.1 204 No Content
Allow: DELETE, GET, HEAD, OPTIONS, POST, PUT
```

- **Implicit HEAD**: Automatically included when GET is registered
- **Always includes OPTIONS**: OPTIONS is always listed as an allowed method
- **Explicit handlers take precedence**: Register your own OPTIONS handler to customize behavior

```go
// Custom OPTIONS handler - takes precedence over auto-generated response
app.OPTIONS("/api/special", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Allow", "GET, POST, CUSTOM")
    w.Header().Set("X-Custom-Header", "value")
    w.WriteHeader(http.StatusNoContent)
}))
```

## Middleware

zerohttp includes a comprehensive set of built-in middlewares:

```go
app := zh.New()

// Add additional middleware (default security middlewares already applied)
app.Use(
    middleware.CORS(config.CORSConfig{
        AllowedOrigins:   []string{"https://example.com"},
        AllowCredentials: true,
    }),
    middleware.RateLimit(config.RateLimitConfig{
        Rate:   100,
        Window: time.Minute,
    }),
)

// Route-specific middleware
app.GET("/admin", adminHandler,
    middleware.BasicAuth(config.BasicAuthConfig{
        Credentials: map[string]string{"admin": "secret"},
    }),
    middleware.RequestBodySize(config.RequestBodySizeConfig{
        MaxBytes: 1024 * 1024, // 1MB limit
    }),
)
```

### Available Middlewares

- **Authentication**: Basic Auth, HMAC Request Signing, JWT Auth
- **Security**: CORS, Security Headers, Request Body Size Limits
- **Rate Limiting**: Rate Limit with configurable algorithms
- **Content Handling**: Compress, Content Charset, Content Encoding, Content Type
- **Monitoring**: Request Logger, Circuit Breaker, Timeout, Recover
- **Utilities**: Request ID, Real IP, Trailing Slash, Set Header, No Cache, With Value

Run `go doc github.com/alexferl/zerohttp/middleware` for detailed documentation on each middleware.

Each middleware accepts a config struct:

```go
// CORS middleware
middleware.CORS(config.CORSConfig{
    AllowedOrigins:   []string{"https://example.com"},
    AllowCredentials: true,
})

// Rate limiting
middleware.RateLimit(config.RateLimitConfig{
    Rate:      50,
    Window:    time.Minute,
    Algorithm: config.TokenBucket,
})

// Compression
middleware.Compress(config.CompressConfig{
    Level: 6,
})

// Security headers with HSTS options
middleware.SecurityHeaders(config.SecurityHeadersConfig{
    CSP:           "default-src 'self'; script-src 'self' 'unsafe-inline'",
    XFrameOptions: "DENY",
    HSTS: config.HSTSConfig{
        MaxAge:  31536000,
        Preload: true,
    },
})
```

## Server Lifecycle Hooks

zerohttp provides lifecycle hooks for both startup and shutdown phases. Hooks allow you to perform initialization before the server starts and cleanup when it stops.

**⚠️ Important:** Hooks **must** respect context cancellation by checking `ctx.Done()`. If a hook blocks without respecting the context, startup/shutdown will hang.

### Startup Hooks

```go
app := zh.New(config.Config{
    Lifecycle: config.LifecycleConfig{
        // Pre-startup: run before servers start (sequential)
        PreStartupHooks: []config.StartupHookConfig{
            {
                Name: "validate-config",
                Hook: func(ctx context.Context) error {
                    return validateConfig()
                },
            },
        },

        // Startup: run with servers starting up (sequential)
        StartupHooks: []config.StartupHookConfig{
            {
                Name: "migrations",
                Hook: func(ctx context.Context) error {
                    return goose.Up(db.DB, "migrations")
                },
            },
        },
    },
})

// Or register programmatically
app.RegisterPreStartupHook("validate-config", func(ctx context.Context) error {
    return validateConfig()
})

app.RegisterStartupHook("migrations", func(ctx context.Context) error {
    return goose.Up(db.DB, "migrations")
})

app.RegisterPostStartupHook("announce-ready", func(ctx context.Context) error {
    return notifyServiceDiscovery()
})
```

### Shutdown Hooks

```go
app := zh.New(config.Config{
    Lifecycle: config.LifecycleConfig{
        // Pre-shutdown: run before servers start shutting down (sequential)
        PreShutdownHooks: []config.ShutdownHookConfig{
            {
                Name: "health",
                Hook: func(ctx context.Context) error {
                    // Mark service as unhealthy to stop receiving traffic
                    health.SetUnhealthy()
                    return nil
                },
            },
        },

        // Shutdown: run concurrently with server shutdown
        ShutdownHooks: []config.ShutdownHookConfig{
            {
                Name: "flush-logs",
                Hook: func(ctx context.Context) error {
                    return logger.Flush()
                },
            },
            {
                Name: "close-db",
                Hook: func(ctx context.Context) error {
                    return db.Close()
                },
            },
        },

        // Post-shutdown: run after all servers are stopped (sequential)
        PostShutdownHooks: []config.ShutdownHookConfig{
            {
                Name: "cleanup",
                Hook: func(ctx context.Context) error {
                    return os.RemoveAll("/tmp/app-*")
                },
            },
        },
    },
})

// Or register programmatically
app.RegisterPreShutdownHook("health", func(ctx context.Context) error {
    health.SetUnhealthy()
    return nil
})

app.RegisterShutdownHook("close-db", func(ctx context.Context) error {
    return db.Close()
})

app.RegisterPostShutdownHook("cleanup", func(ctx context.Context) error {
    return os.RemoveAll("/tmp/app-*")
})
```

### Hook Execution Order

**Startup Phase:**

1. **Pre-startup hooks** - Execute sequentially in registration order
2. **Server startup** - HTTP/HTTPS/HTTP3 servers start concurrently
3. **Startup hooks** - Execute sequentially after pre-startup hooks complete
4. **Post-startup hooks** - Execute sequentially after servers are accepting connections

**Shutdown Phase:**

1. **Pre-shutdown hooks** - Execute sequentially in registration order
2. **Server shutdown** - HTTP/HTTPS/HTTP3 servers shut down concurrently
3. **Shutdown hooks** - Execute concurrently alongside server shutdown
4. **Post-shutdown hooks** - Execute sequentially after all servers are stopped

### Error Handling

- **Startup hooks:** Errors from pre-startup and startup hooks stop server startup
- **Shutdown hooks:** Errors are logged but do not stop the shutdown process
- **Context cancellation:** Context errors (`context.Canceled` or `context.DeadlineExceeded`) abort the current phase early


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
    healthcheck.New(app, healthcheck.DefaultConfig)

    // Or customize endpoints and handlers
    cfg := healthcheck.DefaultConfig
    cfg.LivenessEndpoint = "/health/live"
    cfg.ReadinessEndpoint = "/health/ready"
    cfg.ReadinessHandler = func(w http.ResponseWriter, r *http.Request) error {
        // Check database connections, dependencies, etc.
        if !isAppReady() {
            return zh.Render.Text(w, http.StatusServiceUnavailable, "not ready")
        }
        return zh.Render.Text(w, http.StatusOK, "ready")
    }
    cfg.StartupEndpoint = "/health/startup"
    healthcheck.New(app, cfg)

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
app.Use(middleware.CircuitBreaker(config.CircuitBreakerConfig{
    FailureThreshold:  3,                // Break after 3 failures
    RecoveryTimeout:   10 * time.Second, // Try recovery after 10s
    OpenStatusCode:    503,              // Return 503 when open
}))
```

The circuit breaker operates in three states: **Closed** (normal), **Open** (blocked), and **Half-Open** (testing recovery). It prevents cascading failures when downstream services are unavailable.

## Metrics

zerohttp provides Prometheus-compatible metrics collection with zero external dependencies:

```go
// Metrics are enabled by default and exposed at /metrics
app := zh.New()

// Access metrics in handlers
app.GET("/orders", func(w http.ResponseWriter, r *http.Request) error {
    reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))
    counter := reg.Counter("orders_total", "status")
    counter.WithLabelValues("completed").Inc()
    return zh.Render.JSON(w, http.StatusOK, order)
})
```

Run `go doc github.com/alexferl/zerohttp/metrics` for complete documentation.

## Distributed Tracing

zerohttp includes a pluggable tracing interface for distributed tracing with zero dependencies:

```go
import (
    "github.com/alexferl/zerohttp/middleware"
    "github.com/alexferl/zerohttp/trace"
)

// Implement the Tracer interface (or use OpenTelemetry)
type myTracer struct{}

func (t *myTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
    // Your tracing implementation
    return ctx, span
}

app := zh.New(config.Config{
    Tracer: myTracer,
})

// Add tracing middleware - creates spans for each request
app.Use(middleware.Tracing(myTracer))

// Access spans in handlers
app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    span := trace.SpanFromContext(r.Context())
    span.SetAttributes(trace.String("user.id", "123"))
    return zh.R.JSON(w, 200, zh.M{"message": "ok"})
}))
```

Run `go doc github.com/alexferl/zerohttp/trace` for complete examples including OpenTelemetry integration.

## Static File Serving

Serve static files from embedded filesystems or directories with configurable fallback behavior:

```go
//go:embed static
var staticFiles embed.FS

//go:embed dist
var appFiles embed.FS

app := zh.New()

// API routes
app.GET("/api/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    return zh.Render.JSON(w, 200, zh.M{"status": "healthy"})
}))

// Serve static assets (CSS, JS, images) from embedded FS
app.Files("/static/", staticFiles, "static")

// Serve files from directory (uploads, user content)
app.FilesDir("/uploads/", "./uploads")

// Serve SPA with client-side routing fallback (fallback=true)
app.Static(appFiles, "dist", true, "/api/")

// Serve static website with custom 404 handler (fallback=false)
app.Static(appFiles, "dist", false, "/api/")

// Or serve from directory for development
// app.StaticDir("./dist", true, "/api/")
// app.StaticDir("./dist", false, "/api/")

log.Fatal(app.Start())
```

### Static File Methods

- **`Files(prefix, embedFS, dir)`** - Serves files from embedded FS without fallback (returns 404 for missing files)
- **`FilesDir(prefix, dir)`** - Serves files from directory without fallback (returns 404 for missing files)
- **`Static(embedFS, dir, fallback, apiPrefixes...)`** - Serves web app from embedded FS with configurable fallback:
    - `fallback: true` - Falls back to index.html for missing files (SPA behavior)
    - `fallback: false` - Uses custom NotFound handler for missing files (static site behavior)
- **`StaticDir(dir, fallback, apiPrefixes...)`** - Serves web app from directory with configurable fallback behavior


### Fallback Behavior

**With `fallback: true` (Single Page Applications):**

- Missing files return `index.html` to support client-side routing
- Perfect for React, Vue, Angular apps

**With `fallback: false` (Static Websites):**

- Missing files use your custom `NotFound` handler
- Perfect for traditional static websites with custom 404 pages


### API Prefix Exclusions

The `Static` methods support API prefix exclusions - requests matching specified prefixes return 404 instead of falling back to index.html, allowing API and static routes to coexist cleanly:

```go
// API routes return proper 404s, SPA routes fallback to index.html
app.Static(appFiles, "dist", true, "/api/", "/auth/", "/uploads/")
```

This prevents API endpoints from accidentally serving your SPA's index.html when routes don't exist.

## Extensibility

zerohttp provides multiple ways to customize its behavior through interfaces.

### Extensible Interfaces

Replace core components by implementing interfaces:

```go
// Custom renderer
type MyRenderer struct{}

func (r *MyRenderer) JSON(w http.ResponseWriter, code int, data any) error {
    // Custom JSON rendering logic
    w.Header().Set("X-Custom-JSON", "true")
    w.Header().Set(consts.HeaderContentType, consts.MIMEApplicationJSON)
    w.WriteHeader(code)
    return json.NewEncoder(w).Encode(data)
}

// Replace default
zh.Render = &MyRenderer{}

// Custom binder
type MyBinder struct{}

func (b *MyBinder) JSON(r io.Reader, dst any) error {
    decoder := json.NewDecoder(r)
    decoder.UseNumber() // Use json.Number instead of float64
    return decoder.Decode(dst)
}

func (b *MyBinder) Form(r *http.Request, dst any) error { return nil }
func (b *MyBinder) MultipartForm(r *http.Request, dst any, maxMemory int64) error { return nil }
func (b *MyBinder) Query(r *http.Request, dst any) error { return nil }

// Replace default
zh.Bind = &MyBinder{}

// Custom validator
type MyValidator struct{}

func (v *MyValidator) Struct(dst any) error {
    // Custom validation logic
    return nil
}

func (v *MyValidator) Register(name string, fn func(reflect.Value, string) error) {
    // Custom validator registration
}

// Replace default
zh.Validate = &MyValidator{}
```

### Pluggable Features

Optional features that require external dependencies. Run `go doc` for full documentation and examples.

- **Validator** - External validation libraries (e.g., go-playground/validator)
- **Auto-TLS** - Let's Encrypt automatic certificates
- **HTTP/3** - QUIC support (e.g., quic-go)
- **Server-Sent Events (SSE)** - Real-time streaming
- **WebSocket** - Bidirectional communication (e.g., gorilla/websocket)
- **WebTransport** - HTTP/3 bidirectional streams (e.g., quic-go/webtransport-go)

## Testing Utilities

The `zhtest` package provides fluent, chainable helpers for testing HTTP handlers and middleware. Run `go doc github.com/alexferl/zerohttp/zhtest` for details.

## Profiling

Built-in Go profiling endpoints via `net/http/pprof`, secured by default with auto-generated passwords:

```go
import (
    "log"

    zh "github.com/alexferl/zerohttp"
    "github.com/alexferl/zerohttp/pprof"
)

func main() {
    app := zh.New()

    // Default: auto-generates secure password
    pp := pprof.New(app, pprof.DefaultConfig)
    log.Printf("pprof credentials - username: %s, password: %s", pp.Auth.Username, pp.Auth.Password)

    // With custom credentials
    cfg := pprof.DefaultConfig
    cfg.Auth = &pprof.AuthConfig{
        Username: "admin",
        Password: "secret",
    }
    pp = pprof.New(app, cfg)

    // Disable authentication (not recommended for production)
    cfg = pprof.DefaultConfig
    cfg.Auth = &pprof.AuthConfig{} // empty = disabled
    pp = pprof.New(app, cfg)

    log.Fatal(app.Start())
}
```

Available endpoints at `/debug/pprof/`:

- `/debug/pprof/` - Index page listing all profiles
- `/debug/pprof/profile` - CPU profile (use `?seconds=30`)
- `/debug/pprof/heap` - Memory heap profile
- `/debug/pprof/goroutine` - Goroutine profile
- `/debug/pprof/trace` - Execution trace (use `?seconds=5`)
- `/debug/pprof/block` - Block profile
- `/debug/pprof/mutex` - Mutex profile

Access credentials via the returned `PProf` struct: `pp.Auth.Username`, `pp.Auth.Password`.

## Configuration

zerohttp uses struct-based configuration. Pass a `config.Config` struct to `zh.New()`:

### Server Configuration

```go
app := zh.New(config.Config{
    // Server addresses
    Addr: ":8080", // HTTP server address

    // TLS configuration
    TLS: config.TLSConfig{
        Addr:     ":8443",
        CertFile: "cert.pem",
        KeyFile:  "key.pem",
    },

    // Custom server instances (optional)
    Server: &http.Server{...},     // Custom HTTP server instance
    TLS: config.TLSConfig{
        Server: &http.Server{...},  // Custom HTTPS server instance
    },

    // Custom listeners (optional)
    Listener: &net.TCPListener{...}, // Custom HTTP listener
    TLS: config.TLSConfig{
        Listener: myTLSListener,     // Custom HTTPS listener
    },

    // Logger and validator
    Logger:    myLogger,     // Custom logger instance
    Validator: myValidator,  // Custom struct validator

    // Pluggable features (in Extensions)
    Extensions: config.ExtensionsConfig{
        AutocertManager:    myCertManager,  // Let's Encrypt integration
        HTTP3Server:        myH3Server,     // HTTP/3 server (e.g., quic-go)
        SSEProvider:        mySSEProvider,  // SSE provider for server-sent events
        WebSocketUpgrader:  myWSUpgrader,   // WebSocket upgrader
        WebTransportServer: myWTServer,     // WebTransport server
    },

    // Middleware options
    DisableDefaultMiddlewares: false,                                    // Disable built-in middlewares
    DefaultMiddlewares:        []func(http.Handler) http.Handler{...},  // Custom middleware chain
})
```

### Middleware Configuration

Configure default middlewares directly on the Config struct:

```go
app := zh.New(config.Config{
    RequestID: config.RequestIDConfig{
        Header: "X-Request-ID",
    },
    Recover: config.RecoverConfig{
        StackSize:        4096,
        EnableStackTrace: true,
    },
    RequestBodySize: config.RequestBodySizeConfig{
        MaxBytes: 5 * 1024 * 1024, // 5MB
    },
    SecurityHeaders: config.SecurityHeadersConfig{
        CSP:              "default-src 'self'",
        XFrameOptions:    "DENY",
        HSTS: config.HSTSConfig{
            MaxAge:   31536000,
            Preload:  true,
        },
    },
    RequestLogger: config.RequestLoggerConfig{
        LogErrors: true,
        Fields:    []string{"method", "uri", "status", "duration"},
    },
})
```

### Disabling Default Security

If you need to disable default middlewares:

```go
app := zh.New(config.Config{
    DisableDefaultMiddlewares: true, // Disable all defaults
    // Or provide custom defaults
    DefaultMiddlewares: []func(http.Handler) http.Handler{
        middleware.RequestID(config.DefaultRequestIDConfig),
        middleware.CORS(config.DefaultCORSConfig),
    },
})
```

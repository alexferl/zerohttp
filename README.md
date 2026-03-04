# zerohttp [![Go Report Card](https://goreportcard.com/badge/github.com/alexferl/zerohttp)](https://goreportcard.com/report/github.com/alexferl/zerohttp) [![codecov](https://codecov.io/gh/alexferl/zerohttp/branch/master/graph/badge.svg)](https://codecov.io/gh/alexferl/zerohttp)

**⚠️ This is a pre-v1 release - APIs may change as we work toward a stable v1.0.**

A lightweight HTTP framework for Go built on top of the standard `net/http` library. Designed for simplicity, developer productivity, and security.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Secure by Default](#secure-by-default)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Response Rendering](#response-rendering)
- [Request Binding](#request-binding)
- [Path Parameters](#path-parameters)
- [Query Parameters](#query-parameters)
- [Middleware](#middleware)
- [Route Groups](#route-groups)
- [Static File Serving](#static-file-serving)
    - [Static File Methods](#static-file-methods)
- [Error Handling](#error-handling)
    - [Validation Errors](#validation-errors)
- [Configuration](#configuration)
- [Disabling Default Security](#disabling-default-security)
- [Available Middlewares](#available-middlewares)
- [Extensible Interfaces](#extensible-interfaces)
- [Pluggable Features](#pluggable-features)
    - [Auto-TLS](#auto-tls)
    - [HTTP/3 Support](#http3-support)
    - [WebSocket Support](#websocket-support)
    - [WebTransport Support](#webtransport-support)
- [Health Checks](#health-checks)
- [Circuit Breaker](#circuit-breaker)
- [Graceful Shutdown](#graceful-shutdown)
- [Configuration Reference](#configuration-reference)
    - [Server Configuration](#server-configuration)
    - [Middleware Configuration](#middleware-configuration)
    - [Logging](#logging)


## Features

- **Lightweight**: Built on Go's standard `net/http` with minimal overhead
- **Zero Dependencies**: No external dependencies
- **Secure by Default**: Automatically applies essential security middlewares out of the box
- **Response Rendering**: Built-in support for JSON, HTML, text, and file responses
- **Request Binding**: JSON, form, multipart form, and query parameter parsing with struct tag binding
- **Problem Details**: RFC 9457 Problem Details for HTTP APIs error responses
- **Flexible Routing**: Method-based routing with route groups and parameter support
- **Middleware Support**: Comprehensive middleware system with built-in security, logging, and utility middlewares
- **Built-in Security**: CORS, rate limiting, request body size limits, security headers, and more
- **HTTP/2 & HTTP/3**: Automatic HTTP/2 support for TLS; optional HTTP/3 via pluggable interface
- **Pluggable Architecture**: Extensible interfaces for Auto-TLS, HTTP/3, WebSocket, and WebTransport - bring your own implementations
- **Request Tracing**: Built-in request ID generation and propagation
- **Circuit Breaker**: Prevent cascading failures with configurable circuit breaker middleware
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

> **💡 More Examples:** Check out the [`examples/`](examples/) folder for complete working examples including template rendering, static file serving, middleware usage, advanced configurations and more.


## Response Rendering

Clean, extensible interfaces for all response types:

```go
// JSON responses (most common)
zh.Render.JSON(w, 200, zh.M{"message": "Hello, World!"})

// Text responses
zh.Render.Text(w, 200, "Plain text response")

// HTML responses
zh.Render.HTML(w, 200, "<h1>Welcome</h1>")

// Template rendering with parsed templates
tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.html"))
zh.R.Template(w, 200, tmpl, "index.html", zh.M{"title": "Welcome"})

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

Structured request parsing with validation for JSON, form data, and multipart forms:

### JSON Binding

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

### Form Binding

```go
app.POST("/login", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var form struct {
        Username string   `form:"username"`
        Password string   `form:"password"`
        Remember bool     `form:"remember"`
        Tags     []string `form:"tags"`       // Supports slices
    }

    // Bind application/x-www-form-urlencoded or query parameters
    if err := zh.Bind.Form(r, &form); err != nil {
        return zh.NewProblemDetail(400, err.Error()).Render(w)
    }

    // Process form data...
    return zh.R.JSON(w, 200, zh.M{"user": form.Username})
}))
```

### Multipart Form Binding (File Uploads)

```go
app.POST("/upload", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var form struct {
        Description string            `form:"description"`
        Document    *zh.FileHeader    `form:"document"`    // Single file
        Images      []*zh.FileHeader  `form:"images"`      // Multiple files
    }

    // Bind multipart/form-data with file uploads
    // maxMemory controls how much is stored in memory vs temp files
    if err := zh.Bind.MultipartForm(r, &form, 32<<20); err != nil {
        return zh.NewProblemDetail(400, err.Error()).Render(w)
    }

    // Access uploaded files
    if form.Document != nil {
        file, err := form.Document.Open()
        if err != nil {
            return err
        }
        defer file.Close()

        // Process file...
    }

    return zh.R.JSON(w, 200, zh.M{
        "description": form.Description,
        "files":       len(form.Images),
    })
}))
```

**Short alias available**: Use `zh.B` instead of `zh.Bind` for convenience.

The JSON binder uses `json.Decoder` with `DisallowUnknownFields()` for stricter validation.
Form binding supports automatic type conversion for int, uint, float, bool, and slice types.

## Path Parameters

Type-safe path parameter extraction with generic support:

```go
// Basic string extraction
app.GET("/users/{id}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    id := zh.Param(r, "id")  // Returns string
    return zh.R.JSON(w, 200, zh.M{"user_id": id})
}))

// Typed extraction with error handling
app.GET("/items/{itemID}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    itemID, err := zh.ParamAs[int](r, "itemID")
    if err != nil {
        return zh.R.ProblemDetail(w, zh.NewProblemDetail(400, "Invalid itemID"))
    }
    return zh.R.JSON(w, 200, zh.M{"item_id": itemID})
}))

// Multiple parameters
app.GET("/users/{userID}/posts/{postID}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    userID, _ := zh.ParamAs[int](r, "userID")
    postID, _ := zh.ParamAs[int](r, "postID")
    return zh.R.JSON(w, 200, zh.M{"user_id": userID, "post_id": postID})
}))

// With default value (returns default if param missing or invalid)
app.GET("/products/{category}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    category := zh.ParamOrDefault(r, "category", "all")
    return zh.R.JSON(w, 200, zh.M{"category": category})
}))
```

**Available Functions:**

- `Param(r, "name")` - Extract parameter as string
- `ParamAs[T](r, "name")` - Extract and convert to type T (int, int64, uint, float64, bool, etc.)
- `ParamAsOrDefault[T](r, "name", defaultVal)` - Extract with fallback value
- `ParamOrDefault(r, "name", "default")` - String extraction with fallback

**Supported Types:** `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`


## Query Parameters

Structured query parameter binding with struct tags and type-safe extraction:

```go
// Struct-based binding
app.GET("/search", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req struct {
        Query    string   `query:"q"`
        Category string   `query:"category"`
        Tags     []string `query:"tags"`      // Supports multiple values
        Page     int      `query:"page"`
        Limit    int      `query:"limit"`
        IsActive *bool    `query:"is_active"` // Pointer = optional
    }

    if err := zh.Bind.Query(r, &req); err != nil {
        return zh.NewProblemDetail(400, err.Error()).Render(w)
    }

    // Set defaults
    if req.Page < 1 {
        req.Page = 1
    }
    if req.Limit < 1 {
        req.Limit = 20
    }

    return zh.R.JSON(w, 200, zh.M{
        "query":    req.Query,
        "category": req.Category,
        "tags":     req.Tags,
        "page":     req.Page,
    })
}))

// Embedded structs for reusable pagination
type Pagination struct {
    Page  int `query:"page"`
    Limit int `query:"limit"`
}

type ListRequest struct {
    Pagination        // Embeds page, limit fields
    Search string `query:"search"`
}

app.GET("/items", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req ListRequest
    if err := zh.B.Query(r, &req); err != nil {
        return err
    }
    return zh.R.JSON(w, 200, req)
}))
```

### Individual Parameter Extraction

For simple cases, extract individual parameters with type conversion:

```go
app.GET("/extract", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    // Extract with type conversion and error handling
    userID, err := zh.QueryParamAs[int](r, "user_id")
    if err != nil {
        return zh.NewProblemDetail(400, "Invalid user_id").Render(w)
    }

    // Extract with default value
    page := zh.QueryParamAsOrDefault(r, "page", 1)
    limit := zh.QueryParamAsOrDefault(r, "limit", 20)

    // Simple string extraction
    sort := zh.QueryParam(r, "sort") // Empty string if missing

    return zh.R.JSON(w, 200, zh.M{
        "user_id": userID,
        "page":    page,
        "limit":   limit,
        "sort":    sort,
    })
}))
```

**Available Functions:**

- `Bind.Query(r, &dst)` - Bind all query params to struct with `query` tags
- `QueryParam(r, "name")` - Extract parameter as string
- `QueryParamAs[T](r, "name")` - Extract and convert to type T
- `QueryParamAsOrDefault[T](r, "name", defaultVal)` - Extract with fallback

**Supported Types:** Same as Path Parameters - all primitives, slices, and pointers


## Middleware

zerohttp includes a comprehensive set of built-in middlewares:

```go
app := zh.New()

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
app.Group(func(api zh.Router) {
    api.Use(middleware.RequireAuth())

    api.GET("/users", listUsers)
    api.POST("/users", createUser)
    api.PUT("/users/{id}", updateUser)
    api.DELETE("/users/{id}", deleteUser)
})
```


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
    return zh.R.JSON(w, 200, zh.M{"status": "healthy"})
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


## Error Handling

Built-in support for RFC 9457 Problem Details:

```go
app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    problem := zh.NewProblemDetail(400, "Invalid request")
    problem.Set("field", "email")
    problem.Set("reason", "Email address is required")
    return zh.R.ProblemDetail(w, problem)
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


## Configuration

Flexible configuration system with functional options:

```go
app := zh.New(
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
app := zh.New(
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
- **Utilities**: Request ID, Real IP, Trailing Slash, Set Header, No Cache, With Value

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
zh.Render = &MyRenderer{}

// Custom binder
type MyBinder struct{}

func (b *MyBinder) JSON(r io.Reader, dst any) error {
    // Custom JSON binding logic
    decoder := json.NewDecoder(r)
    decoder.UseNumber() // Use json.Number instead of float64
    return decoder.Decode(dst)
}

func (b *MyBinder) Form(r *http.Request, dst any) error {
    // Custom form binding logic
    return nil
}

func (b *MyBinder) MultipartForm(r *http.Request, dst any, maxMemory int64) error {
    // Custom multipart form binding logic
    return nil
}

func (b *MyBinder) Query(r *http.Request, dst any) error {
    // Custom query parameter binding logic
    return nil
}

// Replace default
zh.Bind = &MyBinder{}
```


## Pluggable Features

zerohttp provides several pluggable features that extend the core functionality through interfaces.
These features require external dependencies and are opt-in:

- **Auto-TLS** - Automatic certificate management (e.g., Let's Encrypt)
- **HTTP/3** - HTTP/3 support over QUIC (HTTP/2 is enabled by default for TLS)
- **WebTransport** - Low-latency bidirectional communication over HTTP/3

Each feature uses a pluggable interface pattern - you bring your own implementation.

### Interface Overview

zerohttp defines minimal interfaces for each pluggable feature. Implement these interfaces
or use existing implementations from the Go ecosystem:

```go
// AutocertManager - Automatic certificate management
interface {
    GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
    HTTPHandler(fallback http.Handler) http.Handler
}

// HTTP3Server - HTTP/3 over QUIC support
interface {
    ListenAndServeTLS(certFile, keyFile string) error
    Shutdown(ctx context.Context) error
    Close() error
}

// WebTransportServer - WebTransport over HTTP/3
interface {
    ListenAndServeTLS(certFile, keyFile string) error
    Close() error
}
```

All pluggable features are configured via functional options:

```go
app := zh.New(
    config.WithAutocertManager(myCertManager),
    config.WithHTTP3Server(myH3Server),
    config.WithWebTransportServer(myWTServer),
)
```

### Auto-TLS

AutoTLS provides automatic certificate management via a pluggable interface. You can use
[golang.org/x/crypto/acme/autocert](https://pkg.go.dev/golang.org/x/crypto/acme/autocert)
for Let's Encrypt, or any other provider that implements the `AutocertManager` interface:

```go
import (
    "golang.org/x/crypto/acme/autocert"
)

// Create autocert manager (implements config.AutocertManager interface)
manager := &autocert.Manager{
    Cache:      autocert.DirCache("/tmp/certs"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("example.com", "www.example.com"),
}

app := zh.New(
    config.WithAutocertManager(manager),
)

// StartAutoTLS starts HTTP (for ACME challenges) and HTTPS servers
app.StartAutoTLS()
```


### HTTP/3 Support

zerohttp supports HTTP/3 through a pluggable interface. Users can inject their own HTTP/3
implementation (e.g., [quic-go/http3](https://github.com/quic-go/quic-go)).

#### Basic HTTP/3 with TLS certificates

```go
import "github.com/quic-go/quic-go/http3"

// Create your zerohttp server
app := zh.New()

// Add routes
app.GET("/", handler)

// Create HTTP/3 server using quic-go
h3Server := &http3.Server{
    Addr:    ":443",
    Handler: app,
}

// Inject the HTTP/3 server
app.SetHTTP3Server(h3Server)

// Start HTTPS - HTTP/3 starts automatically!
app.StartTLS("cert.pem", "key.pem")
```


### WebSocket Support

WebSocket provides real-time bidirectional communication over TCP. zerohttp supports WebSocket
through a pluggable interface - you bring your own WebSocket library (e.g., [gorilla/websocket](https://github.com/gorilla/websocket),
[nhooyr/websocket](https://github.com/nhooyr/websocket)).

```go
import (
    "github.com/gorilla/websocket"
    zh "github.com/alexferl/zerohttp"
    "github.com/alexferl/zerohttp/config"
)

// Wrap gorilla/websocket to implement config.WebSocketUpgrader
type myUpgrader struct {
    upgrader *websocket.Upgrader
}

func (m *myUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (config.WebSocketConn, error) {
    conn, err := m.upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, err
    }
    return &myConn{conn: conn}, nil
}

type myConn struct {
    conn *websocket.Conn
}

func (c *myConn) ReadMessage() (int, []byte, error)  { return c.conn.ReadMessage() }
func (c *myConn) WriteMessage(mt int, data []byte) error { return c.conn.WriteMessage(mt, data) }
func (c *myConn) Close() error                        { return c.conn.Close() }
func (c *myConn) RemoteAddr() net.Addr               { return c.conn.RemoteAddr() }

func main() {
    // Create server with WebSocket support
    gupgrader := &websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    app := zh.New(
        config.WithWebSocketUpgrader(&myUpgrader{upgrader: gupgrader}),
    )

    // WebSocket endpoint
    app.GET("/ws", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        ws, err := app.WebSocketUpgrader().Upgrade(w, r)
        if err != nil {
            return err
        }
        defer ws.Close()

        // Echo loop
        for {
            mt, msg, err := ws.ReadMessage()
            if err != nil {
                break
            }
            if err := ws.WriteMessage(mt, msg); err != nil {
                break
            }
        }
        return nil
    }))

    app.ListenAndServe()
}
```

> **💡 Complete Example:** See [`examples/websocket/`](examples/websocket/) for a full working server with HTML client.


### WebTransport Support

WebTransport provides low-latency, bidirectional communication over HTTP/3:

```go
import (
    "github.com/quic-go/quic-go/http3"
    webtransport "github.com/quic-go/webtransport-go"
)

app := zh.New()

// Create HTTP/3 server
h3 := &http3.Server{Addr: ":8443", Handler: app}

// Create WebTransport server
wtServer := &webtransport.Server{
    H3:          h3,
    CheckOrigin: func(r *http.Request) bool { return true },
}
webtransport.ConfigureHTTP3Server(h3)

// Set WebTransport server - zerohttp starts it automatically
app.SetWebTransportServer(wtServer)

// Register WebTransport endpoint
app.CONNECT("/wt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    sess, err := wtServer.Upgrade(w, r)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    go handleSession(sess) // Handle streams/datagrams
}))

app.ListenAndServeTLS("cert.pem", "key.pem")
```

> **💡 Complete Example:** See [`examples/webtransport/`](examples/webtransport/) for a full working server with HTML client.


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


## Graceful Shutdown

zerohttp provides graceful shutdown hooks for cleanup tasks during server shutdown. Hooks are called during `Shutdown()` and allow you to perform cleanup like closing database connections, flushing logs, and notifying external systems.

**⚠️ Important:** Hooks **must** respect context cancellation by checking `ctx.Done()`. If a hook blocks without respecting the context, shutdown will hang.

```go
app := zh.New(
    // Pre-shutdown: run before servers start shutting down (sequential)
    config.WithPreShutdownHook("health", func(ctx context.Context) error {
        // Mark service as unhealthy to stop receiving traffic
        health.SetUnhealthy()
        return nil
    }),

    // Shutdown: run concurrently with server shutdown
    config.WithShutdownHook("flush-logs", func(ctx context.Context) error {
        return logger.Flush()
    }),
    config.WithShutdownHook("close-db", func(ctx context.Context) error {
        // Always check context cancellation for long operations
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            return db.Close()
        }
    }),

    // Post-shutdown: run after all servers are stopped (sequential)
    config.WithPostShutdownHook("cleanup", func(ctx context.Context) error {
        return os.RemoveAll("/tmp/app-*")
    }),
)

// Hooks can also be registered programmatically
app.RegisterShutdownHook("metrics", func(ctx context.Context) error {
    return metrics.Push(ctx, gateway)
})
```

### Hook Execution Order

1. **Pre-shutdown hooks** - Execute sequentially in registration order
2. **Server shutdown** - HTTP/HTTPS/HTTP3 servers shut down concurrently
3. **Shutdown hooks** - Execute concurrently alongside server shutdown
4. **Post-shutdown hooks** - Execute sequentially in registration order

Hook errors are logged but do not stop the shutdown process. Context errors (`context.Canceled` or `context.DeadlineExceeded`) from pre-shutdown hooks will abort shutdown early.


## Configuration Reference

The functional options pattern provides structured configuration for all aspects of the server:

### Server Configuration

- `config.WithAddr()` - HTTP server address
- `config.WithTLSAddr()` - HTTPS server address
- `config.WithServer()` - Custom HTTP server instance
- `config.WithTLSServer()` - Custom HTTPS server instance
- `config.WithListener()` - Custom HTTP listener
- `config.WithTLSListener()` - Custom HTTPS listener
- `config.WithCertFile()` - TLS certificate file path
- `config.WithKeyFile()` - TLS key file path
- `config.WithAutocertManager()` - Let's Encrypt integration
- `config.WithHTTP3Server()` - HTTP/3 server (e.g., quic-go)
- `config.WithWebSocketUpgrader()` - WebSocket upgrader (e.g., gorilla/websocket)
- `config.WithWebTransportServer()` - WebTransport server (e.g., webtransport-go)
- `config.WithPreShutdownHook()` - Hook to run before server shutdown
- `config.WithShutdownHook()` - Hook to run concurrently with server shutdown
- `config.WithPostShutdownHook()` - Hook to run after server shutdown


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

# zerohttp [![Go Reference](https://pkg.go.dev/badge/github.com/alexferl/zerohttp.svg)](https://pkg.go.dev/github.com/alexferl/zerohttp) [![Go Report Card](https://goreportcard.com/badge/github.com/alexferl/zerohttp)](https://goreportcard.com/report/github.com/alexferl/zerohttp) [![Coverage Status](https://coveralls.io/repos/github/alexferl/zerohttp/badge.svg?branch=master)](https://coveralls.io/github/alexferl/zerohttp?branch=master)

A lightweight, secure-by-default HTTP framework for Go. Built on `net/http` with zero external dependencies.

## Why zerohttp?

**Built on stdlib, not instead of it.** zerohttp builds on Go's `net/http` rather than replacing it, so your handlers stay standard `http.HandlerFunc` and work with existing middleware and tooling.

**Secure by default.** Sensible security headers, request body limits, panic recovery, and request IDs are applied automatically for every request.

**Zero dependencies.** Single module, standard library only, so your service stays lean and easy to upgrade.

**Handler errors that make sense.** Handlers return `error`, and RFC 9457 Problem Details responses are generated for you automatically.

## Features

- **Zero dependencies** - Single module, no external deps
- **Secure by default** - Security headers, body limits, recovery, request IDs enabled automatically
- **Standard library foundation** - Built on `net/http`, works with any `http.Handler` middleware
- **Handler errors** - Return `error`, get proper HTTP responses automatically
- **Request binding** - JSON, form, multipart, and query params to structs with struct tags
- **Validation** - Built-in struct validation with 40+ validators
- **Problem Details** - RFC 9457 compliant error responses
- **Middleware** - CORS, rate limiting, auth, circuit breaker, and more
- **Metrics** - Prometheus-compatible metrics at `/metrics`
- **Lifecycle hooks** - Pre/post startup and shutdown hooks
- **Pluggable** - Bring your own validator, tracer, HTTP/3, WebSocket, SSE

## Installation

```shell
go get github.com/alexferl/zerohttp
```

Requires Go 1.25 or later.

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    zh "github.com/alexferl/zerohttp"
)

func main() {
    app := zh.New()

    app.GET("/hello/{name}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        name := zh.Param(r, "name")
        return zh.Render.JSON(w, http.StatusOK, zh.M{"message": "Hello, " + name + "!"})
    }))

    log.Fatal(app.Start())
}
```

```shell
go run main.go
curl http://localhost:8080/hello/world
{"message":"Hello, world!"}
```

## Examples

See the [`examples/`](examples/) directory for more complete examples.

### Request Binding & Validation

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
}

app.POST("/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req CreateUserRequest
    if err := zh.BindAndValidate(r, &req); err != nil {
        return err // Automatic Problem Details response
    }
    // Process valid request...
    return zh.R.JSON(w, http.StatusCreated, req)
}))
```

### Route Groups with Middleware

```go
app.Group(func(api zh.Router) {
    api.Use(basicauth.New(basicauth.Config{
        Credentials: map[string]string{"admin": "secret"},
    }))
    api.GET("/admin/dashboard", dashboardHandler)
})
```

### Query Parameters

```go
type SearchRequest struct {
    Query string `query:"q" validate:"required"`
    Limit int    `query:"limit" validate:"max=100"`
}

app.GET("/search", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req SearchRequest
    if err := zh.BindAndValidate(r, &req); err != nil {
        return err
    }
    return zh.R.JSON(w, http.StatusOK, zh.M{"results": []string{}})
}))
```

### Error Handling

```go
app.GET("/users/{id}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    id := zh.Param(r, "id")
    user, err := db.GetUser(id)
    if err != nil {
        return zh.NewProblemDetail(http.StatusNotFound, "user not found").Render(w)
    }
    return zh.R.JSON(w, http.StatusOK, user)
}))
```

### Response Helpers

```go
app.GET("/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    return zh.R.Text(w, http.StatusOK, "healthy")
}))

app.GET("/docs", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    return zh.R.Redirect(w, r, "https://pkg.go.dev/github.com/alexferl/zerohttp", http.StatusFound)
}))
```

## Configuration

zerohttp uses struct-based configuration:

```go
app := zh.New(config.Config{
    Addr: ":8080",
    TLS: config.TLSConfig{
        Addr:     ":8443",
        CertFile: "cert.pem",
        KeyFile:  "key.pem",
    },
    RequestBodySize: config.RequestBodySizeConfig{
        MaxBytes: 5 * 1024 * 1024, // 5MB
    },
})
```

## Secure by Default

These middlewares are applied automatically:

- **Request ID** - Unique IDs for tracing
- **Panic Recovery** - Graceful panic handling with stack traces
- **Request Body Size Limits** - DoS protection (1MB default)
- **Security Headers** - CSP, HSTS, X-Frame-Options, etc.
- **Request Logging** - Structured request/response logging

Disable or customize via `config.Config`.

## Testing

The `zhtest` package provides fluent test helpers:

```go
func TestGetUser(t *testing.T) {
    app := setupRouter()
    req := zhtest.NewRequest(http.MethodGet, "/users/123").Build()
    w := zhtest.Serve(app, req)
    zhtest.AssertWith(t, w).Status(http.StatusOK).JSONPathEqual("name", "John")
}
```

## License

MIT License - see [LICENSE](LICENSE) for details.

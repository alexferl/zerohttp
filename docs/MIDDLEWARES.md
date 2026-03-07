# Middleware

zerohttp provides a comprehensive set of middleware for security, authentication, rate limiting, and request handling.

## Table of Contents

- [Overview](#overview)
- [Using Middleware](#using-middleware)
- [Available Middlewares](#available-middlewares)
  - [Authentication](#authentication)
    - [Basic Auth](#basic-auth)
    - [HMAC Request Signing](#hmac-request-signing)
  - [Security](#security)
  - [Rate Limiting](#rate-limiting)
  - [Content Handling](#content-handling)
  - [Monitoring](#monitoring)
  - [Utilities](#utilities)

## Overview

Middleware can be applied at the application level (affects all routes) or route level (affects specific routes).

## Using Middleware

```go
// Application-level middleware (applies to all routes)
app.Use(middleware.CORS(config.DefaultCORSConfig))

// Route group middleware
app.Group(func(api zh.Router) {
    api.Use(middleware.BasicAuth(config.BasicAuthConfig{
        Credentials: map[string]string{"admin": "secret"},
    }))
    api.GET("/admin", adminHandler)
})

// Route-specific middleware
app.GET("/admin", adminHandler,
    middleware.BasicAuth(config.BasicAuthConfig{
        Credentials: map[string]string{"admin": "secret"},
    }),
)
```

## Available Middlewares

### Authentication

#### Basic Auth

HTTP Basic Authentication with configurable credentials and realm.

```go
middleware.BasicAuth(config.BasicAuthConfig{
    Credentials: map[string]string{
        "admin": "secret-password",
    },
    Realm: "Restricted Area",
    ExemptPaths: []string{"/health", "/metrics"},
})
```

#### HMAC Request Signing

Stateless machine-to-machine API authentication using HMAC-SHA256 request signing. Inspired by AWS Signature Version 4.

```go
middleware.HMACAuth(config.HMACAuthConfig{
    CredentialStore: func(accessKeyID string) []string {
        // Return one or more secrets (supports key rotation)
        return credentials[accessKeyID]
    },
    MaxSkew:         5 * time.Minute,  // Max time difference for replay protection
    ClockSkewGrace:  1 * time.Minute,  // Additional tolerance for clock drift
    RequiredHeaders: []string{"host", "x-timestamp"},
    OptionalHeaders: []string{"content-type"},
    ExemptPaths:     []string{"/health", "/metrics"},
    AllowUnsignedPayload: false,       // Set true for streaming/large bodies
    MaxBodySize:     10 * 1024 * 1024, // 10MB max body size
    AllowPresignedURLs: true,          // Enable presigned URL support
})
```

**Features:**

- **Key Rotation**: Supports multiple valid secrets per access key ID
- **Replay Protection**: Timestamp validation with configurable window
- **Body Integrity**: Optional body hash verification
- **Pre-signed URLs**: Share URLs with embedded authentication
- **Multiple Algorithms**: SHA256, SHA384, SHA512

**Client Usage:**

```go
// Create a signer
signer := middleware.NewHMACSigner("access-key-id", "secret-key")

// Sign a request
req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
err := signer.SignRequest(req)

// Make the request
resp, err := http.DefaultClient.Do(req)
```

**Pre-signed URLs:**

```go
// Create a URL valid for 5 minutes
req, _ := http.NewRequest("GET", "https://api.example.com/download/file.zip", nil)
presignedURL, err := signer.PresignURL(req, 5*time.Minute)

// Share the presignedURL - anyone with the URL can access without headers
// https://api.example.com/download/file.zip?X-HMAC-Algorithm=HMAC-SHA256&X-HMAC-Credential=...
```

### Security

#### CORS

Cross-Origin Resource Sharing configuration:

```go
middleware.CORS(config.CORSConfig{
    AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:          86400,
})
```

#### Security Headers

Sets essential security headers:

```go
middleware.SecurityHeaders(config.SecurityHeadersConfig{
    CSP:           "default-src 'self'; script-src 'self'",
    XFrameOptions: "DENY",
    HSTS: config.HSTSConfig{
        MaxAge:  31536000,
        Preload: true,
    },
})
```

#### Request Body Size

Limits request body size to prevent DoS attacks:

```go
middleware.RequestBodySize(config.RequestBodySizeConfig{
    MaxBytes: 1024 * 1024, // 1MB limit
})
```

### Rate Limiting

Token bucket or sliding window rate limiting:

```go
middleware.RateLimit(config.RateLimitConfig{
    Rate:      100,
    Window:    time.Minute,
    Algorithm: config.TokenBucket, // or config.SlidingWindow
})
```

### Content Handling

#### Compression

Gzip compression with configurable level:

```go
middleware.Compress(config.CompressConfig{
    Level:     6,
    Types:     []string{"text/html", "application/json"},
    MinLength: 1024,
})
```

#### Content Type

Enforces Content-Type header:

```go
middleware.ContentType(config.ContentTypeConfig{
    Allowed: []string{"application/json", "application/xml"},
})
```

### Monitoring

#### Request Logger

Structured request/response logging:

```go
middleware.RequestLogger(logger, config.RequestLoggerConfig{
    Fields: []string{"method", "path", "status", "duration"},
})
```

#### Circuit Breaker

Prevents cascading failures:

```go
middleware.CircuitBreaker(config.CircuitBreakerConfig{
    FailureThreshold:    5,
    RecoveryTimeout:     30 * time.Second,
    SuccessThreshold:    3,
})
```

#### Timeout

Request timeout handling:

```go
middleware.Timeout(config.TimeoutConfig{
    Duration: 30 * time.Second,
})
```

### Utilities

#### Request ID

Generates unique request IDs:

```go
middleware.RequestID(config.RequestIDConfig{
    Header: "X-Request-ID",
})
```

#### Real IP

Extracts client IP from headers:

```go
middleware.RealIP(config.RealIPConfig{
    Headers: []string{"X-Forwarded-For", "X-Real-IP"},
})
```

#### Trailing Slash

Normalizes trailing slashes:

```go
middleware.TrailingSlash(config.TrailingSlashConfig{
    Redirect: true,
    Status:   http.StatusMovedPermanently,
})
```

## Creating Custom Middleware

Middleware follows the standard Go `http.Handler` interface:

```go
func MyMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Do something before the handler

            next.ServeHTTP(w, r)

            // Do something after the handler
        })
    }
}
```

## Middleware Execution Order

Middleware is executed in the order it's added:

```go
app.Use(middleware.A) // First
app.Use(middleware.B) // Second

// Execution order: A → B → Handler → B (after) → A (after)
```

For route-level middleware:

```go
app.GET("/path", handler, middleware.A, middleware.B)
// Execution order: A → B → Handler → B (after) → A (after)
```

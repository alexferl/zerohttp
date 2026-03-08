# Middleware

zerohttp provides a comprehensive set of middleware for security, authentication, rate limiting, and request handling.

## Table of Contents

- [Overview](#overview)
- [Using Middleware](#using-middleware)
- [Available Middlewares](#available-middlewares)
  - [Authentication](#authentication)
    - [Basic Auth](#basic-auth)
    - [JWT Authentication](#jwt-authentication)
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

#### JWT Authentication

JSON Web Token (JWT) authentication with pluggable `TokenStore` interface. Includes a built-in HS256 implementation (zero dependencies) or bring your own JWT library.

```go
// Using built-in HS256 (zero dependencies)
hp := middleware.HS256Options{
    Secret: []byte("your-secret-key"),
    Issuer: "my-app",
}

jwtCfg := config.JWTAuthConfig{
    TokenStore:     middleware.NewHS256TokenStore(hp.Secret, hp),
    RequiredClaims: []string{"sub"},
    ExemptPaths:    []string{"/login", "/register"},
}

app.Use(middleware.JWTAuth(jwtCfg))
```

**TokenStore Interface:**

Users implement the `TokenStore` interface to integrate their preferred JWT library:

```go
type TokenStore interface {
    // Validate parses and validates a JWT token
    Validate(token string) (JWTClaims, error)

    // Generate creates a new signed JWT token
    Generate(claims JWTClaims, tokenType TokenType) (string, error)

    // Revoke invalidates a refresh token (called during logout)
    Revoke(claims JWTClaims) error

    // IsRevoked checks if a refresh token has been revoked
    IsRevoked(claims JWTClaims) bool
}
```

**Token Generation:**

```go
// In your login handler
claims := middleware.HS256Claims{
    "sub":   userID,
    "scope": "read write",
}

accessToken, _ := middleware.GenerateAccessToken(r, claims, cfg)
refreshToken, _ := middleware.GenerateRefreshToken(r, claims, cfg)
```

**Refresh Token Endpoint:**

```go
// Built-in refresh handler (calls TokenStore.IsRevoked to check revocation)
app.POST("/auth/refresh", middleware.RefreshTokenHandler(jwtCfg))
```

**Logout Endpoint:**

Use the built-in logout handler to revoke refresh tokens:

```go
// TokenStore.Revoke is called during logout
app.POST("/auth/logout", middleware.LogoutTokenHandler(jwtCfg))
```

**Security Features:**

- Revoked tokens are blocked immediately on all requests (not just at refresh)
- Refresh tokens cannot be used as access tokens for protected endpoints
- Constant-time signature verification to prevent timing attacks

**Accessing Claims in Handlers:**

```go
func profileHandler(w http.ResponseWriter, r *http.Request) error {
    jwt := middleware.GetJWTClaims(r)
    subject := jwt.Subject()     // Get 'sub' claim
    scopes := jwt.Scopes()       // Get 'scope' claim as []string

    if !jwt.HasScope("admin") {
        return zh.R.JSON(w, http.StatusForbidden, zh.M{"error": "admin required"})
    }

    return zh.R.JSON(w, http.StatusOK, zh.M{
        "subject": subject,
        "scopes":  scopes,
    })
}
```

**With External JWT Library:**

```go
// Using golang-jwt/jwt (bring your own)
type MyTokenStore struct {
    secret []byte
}

func (s *MyTokenStore) Validate(token string) (config.JWTClaims, error) {
    return jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
        return s.secret, nil
    })
}

func (s *MyTokenStore) Generate(claims config.JWTClaims, tokenType config.TokenType) (string, error) {
    // Your signing logic
}

func (s *MyTokenStore) Revoke(claims config.JWTClaims) error {
    // Store jti in Redis/DB
    return nil
}

func (s *MyTokenStore) IsRevoked(claims config.JWTClaims) bool {
    // Check jti in Redis/DB
    return false
}

jwtCfg := config.JWTAuthConfig{
    TokenStore:     &MyTokenStore{secret: secret},
    RequiredClaims: []string{"sub"},
}

app.Use(middleware.JWTAuth(jwtCfg))
```

**Custom Token Extraction:**

By default, tokens are extracted from the `Authorization: Bearer <token>` header. You can customize this with a `TokenExtractor` function:

```go
// Extract from cookie
jwtCfg := config.JWTAuthConfig{
    TokenStore: store,
    TokenExtractor: func(r *http.Request) string {
        if cookie, err := r.Cookie("jwt"); err == nil {
            return cookie.Value
        }
        return ""
    },
}

// Extract from custom header
jwtCfg := config.JWTAuthConfig{
    TokenStore: store,
    TokenExtractor: func(r *http.Request) string {
        return r.Header.Get("X-API-Token")
    },
}

// Try cookie first, then fallback to Authorization header
// Useful for supporting both browser (cookie) and API (Bearer) clients
jwtCfg := config.JWTAuthConfig{
    TokenStore: store,
    TokenExtractor: func(r *http.Request) string {
        if cookie, err := r.Cookie("jwt"); err == nil && cookie.Value != "" {
            return cookie.Value
        }
        // Fallback to Authorization header
        auth := r.Header.Get("Authorization")
        if strings.HasPrefix(auth, "Bearer ") {
            return strings.TrimPrefix(auth, "Bearer ")
        }
        return ""
    },
}
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

# zerohttp Roadmap

This document tracks potential features to add to zerohttp. All additions must maintain the **zero external dependencies** constraint.

## Planned

Features actively being worked on or planned for next release.

_None currently._

---

## Under Consideration

### High Priority

- **Metrics Middleware** - Built-in Prometheus-compatible `/metrics` endpoint (zero dependencies). Includes public registry API so middlewares (circuit breaker, rate limiter) and user code can expose their own metrics.

### Medium Priority

_None currently._

### Lower Priority

_None currently._

---

## Completed

- [x] HTTP/HTTPS server with graceful shutdown
- [x] Middleware system with chain support
- [x] Router with method-based routing and groups
- [x] JSON request/response binding and rendering
- [x] RFC 9457 Problem Details support
- [x] Static file serving (SPA and traditional)
- [x] Health check endpoints
- [x] Circuit breaker middleware
- [x] HTTP/3 support (pluggable interface)
- [x] WebTransport support (pluggable interface)
- [x] Auto-TLS support (pluggable interface with Let's Encrypt)
- [x] Comprehensive security middleware (CORS, security headers, rate limiting, body size limits)
- [x] Structured logging interface
- [x] Form/Multipart Binding
- [x] Path Parameter Extraction
- [x] Query Parameter Binding
- [x] ETag Middleware
- [x] CSRF Middleware
- [x] Graceful Shutdown Hooks
- [x] WebSocket Support (pluggable interface)
- [x] Server-Sent Events (SSE) with event replay and broadcast hub
- [x] Testing Utilities Package
- [x] Reverse Proxy Middleware
- [x] Request Validation
- [x] API Consistency
- [x] pprof Endpoints
- [x] HMAC Request Signing Middleware
- [x] JWT Authentication Middleware

---

## Principles

1. **Zero Dependencies** - Use only Go standard library
2. **Pluggable Design** - External features (HTTP/3, WebTransport, Auto-TLS) use interfaces
3. **Secure by Default** - Apply security best practices automatically
4. **Simple API** - Prefer clean, minimal interfaces

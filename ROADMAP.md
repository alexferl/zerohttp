# zerohttp Roadmap

This document tracks potential features to add to zerohttp. All additions must maintain the **zero external dependencies** constraint.

## Planned

Features actively being worked on or planned for next release.

_None currently._

---

## Under Consideration

### High Priority

- [ ] **WebSocket Support** - Native `net/http` upgrade handling for RFC 6455 WebSocket (not just WebTransport). Can be implemented with stdlib `crypto/sha1` and `encoding/base64`.

### Medium Priority

- [ ] **Server-Sent Events (SSE)** - Pure stdlib SSE helper using proper headers and chunked encoding.

- [x] **CSRF Middleware** - Double-submit cookie pattern using `crypto/rand`, `crypto/hmac`, `crypto/subtle`, and `encoding/base64`.

- [ ] **Request Validation** - Struct validation using `reflect` to read struct tags, with custom validation functions (no external libs).

- [ ] **Reverse Proxy Middleware** - Wrapper around `net/http/httputil.ReverseProxy` (stdlib) for proxying to upstream servers.

### Lower Priority

- [ ] **Graceful Shutdown Hooks** - Extend existing shutdown with user-defined callback functions for cleanup tasks.

- [ ] **Testing Utilities Package** - Test helpers, `httptest.ResponseRecorder` wrappers, handler assertion helpers.

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
- [x] Form/Multipart Binding - `Bind.Form()` and `Bind.MultipartForm()` with struct tag binding
- [x] Path Parameter Extraction - Type-safe `ParamAs[T]()` generics and helper functions
- [x] Query Parameter Binding - Structured binding of query params to structs using `reflect`
- [x] ETag Middleware - Automatic ETag generation using stdlib `hash/fnv` or `crypto/md5`, with `If-None-Match` handling for 304 responses

---

## Principles

1. **Zero Dependencies** - Use only Go standard library
2. **Pluggable Design** - External features (HTTP/3, WebTransport, Auto-TLS) use interfaces
3. **Secure by Default** - Apply security best practices automatically
4. **Simple API** - Prefer clean, minimal interfaces

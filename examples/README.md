# zerohttp Examples

This directory contains example applications demonstrating various features of zerohttp.

## Directory Structure

Examples are organized by category:

### `core/` - Core Framework Features

Basic functionality that doesn't require external dependencies beyond the standard library.

- [**`binding_form/`**](core/binding_form/) - Form data binding
- [**`binding_param/`**](core/binding_param/) - URL parameter binding
- [**`binding_query/`**](core/binding_query/) - Query parameter binding
- [**`crud/`**](core/crud/) - CRUD operations with in-memory store
- [**`custom_error_handlers/`**](core/custom_error_handlers/) - Custom error handling
- [**`docker/`**](core/docker/) - Docker containerization example (has go.mod)
- [**`file_server/`**](core/file_server/) - Static file serving
- [**`file_upload/`**](core/file_upload/) - Multipart file uploads
- [**`graceful/`**](core/graceful/) - Graceful shutdown handling
- [**`hsts/`**](core/hsts/) - HTTP Strict Transport Security
- [**`hello_world/`**](core/hello_world/) - Simplest possible server
- [**`lifecycle/`**](core/lifecycle/) - Server lifecycle hooks
- [**`logger/`**](core/logger/) - Log level filtering
- [**`problem_detail/`**](core/problem_detail/) - RFC 7807 Problem Detail responses
- [**`rendering/`**](core/rendering/) - Response rendering methods (JSON, HTML, text, blob, stream)
- [**`request_tracing/`**](core/request_tracing/) - Request ID propagation for tracing
- [**`route_groups/`**](core/route_groups/) - Route groups with nested middleware
- [**`static_spa/`**](core/static_spa/) - Single Page Application serving
- [**`static_website/`**](core/static_website/) - Static website serving
- [**`template/`**](core/template/) - HTML template rendering
- [**`template_renderer/`**](core/template_renderer/) - Custom template renderer setup
- [**`testing/`**](core/testing/) - Testing handlers with zhtest
- [**`tls/`**](core/tls/) - HTTPS/TLS configuration
- [**`validation_basic/`**](core/validation_basic/) - Basic struct validation
- [**`validation_custom/`**](core/validation_custom/) - Custom validation rules
- [**`validation_goplayground/`**](core/validation_goplayground/) - go-playground/validator integration
- [**`validation_nested/`**](core/validation_nested/) - Nested struct validation

### `middleware/` - Middleware Examples

Demonstrations of built-in and custom middleware.

- [**`basicauth/`**](middleware/basic_auth/) - Basic authentication
- [**`cache/`**](middleware/cache/) - HTTP caching middleware
- [**`cache_redis/`**](middleware/cache_redis/) - Redis-backed HTTP caching (has go.mod)
- [**`circuitbreaker/`**](middleware/circuitbreaker/) - Circuit breaker pattern
- [**`compress/`**](middleware/compress/) - Compression middleware (gzip/deflate)
- [**`compress_brotli/`**](middleware/compress_brotli/) - Brotli compression (has go.mod)
- [**`compress_zstd/`**](middleware/compress_zstd/) - Zstd compression (has go.mod)
- [**`contentcharset/`**](middleware/contentcharset/) - Content charset validation
- [**`contentencoding/`**](middleware/contentencoding/) - Content encoding validation
- [**`contenttype/`**](middleware/contenttype/) - Content type validation/middleware
- [**`cors/`**](middleware/cors/) - CORS handling
- [**`csrf/`**](middleware/csrf/) - CSRF protection
- [**`custom/`**](middleware/custom/) - Writing custom middleware
- [**`etag/`**](middleware/etag/) - ETag generation
- [**`hmacauth/`**](middleware/hmacauth/) - HMAC request signing
- [**`hostvalidation/`**](middleware/host/) - Host header validation
- [**`idempotency/`**](middleware/idempotency/) - Idempotent request handling
- [**`idempotency_redis/`**](middleware/idempotency_redis/) - Redis-backed idempotent request handling (has go.mod)
- [**`jwtauth/`**](middleware/jwtauth/) - JWT authentication
- [**`jwtauth_golang_jwt/`**](middleware/jwtauth_golang_jwt/) - golang-jwt integration (has go.mod)
- [**`jwtauth_lestrrat_jwx/`**](middleware/jwtauth_lestrrat_jwx/) - lestrrat-go/jwx integration (has go.mod)
- [**`jwtauth_refresh/`**](middleware/jwtauth_refresh/) - JWT refresh token flow (has go.mod)
- [**`nocache/`**](middleware/nocache/) - Cache control headers
- [**`ratelimit/`**](middleware/ratelimit/) - Rate limiting (in-memory)
- [**`ratelimit_redis/`**](middleware/ratelimit_redis/) - Redis-backed rate limiting (has go.mod)
- [**`realip/`**](middleware/realip/) - Real IP extraction
- [**`recover/`**](middleware/recover/) - Panic recovery
- [**`requestbodysize/`**](middleware/requestbodysize/) - Request body size limits
- [**`requestid/`**](middleware/requestid/) - Request ID generation
- [**`requestlogger/`**](middleware/requestlogger/) - Request logging
- [**`reverseproxy/`**](middleware/reverseproxy/) - Reverse proxy setup
- [**`securityheaders/`**](middleware/securityheaders/) - Security headers
- [**`securityheaders_nonce/`**](middleware/securityheaders_nonce/) - CSP with nonces
- [**`setheader/`**](middleware/setheader/) - Header manipulation
- [**`timeout/`**](middleware/timeout/) - Request timeouts
- [**`tracer/`**](middleware/tracer/) - Distributed tracing
- [**`tracer_jaeger/`**](middleware/tracer_jaeger/) - Jaeger tracing (has go.mod)
- [**`tracer_otel/`**](middleware/tracer_otel/) - OpenTelemetry tracing (has go.mod)
- [**`trailingslash/`**](middleware/trailingslash/) - Trailing slash handling
- [**`value/`**](middleware/value/) - Context value storage

### `extensions/` - Extension Points

Examples showing how to integrate with external protocols and features via config extensions.

- [**`autocert/`**](extensions/autocert/) - Automatic TLS via Let's Encrypt (has go.mod)
- [**`http3/`**](extensions/http3/) - HTTP/3 and QUIC support (has go.mod)
- [**`http3_autocert/`**](extensions/http3_autocert/) - HTTP/3 with AutoTLS (has go.mod)
- [**`websocket/`**](extensions/websocket/) - WebSocket support (has go.mod)
- [**`webtransport/`**](extensions/webtransport/) - WebTransport protocol (has go.mod)
- [**`webstransport_autocert/`**](extensions/webstransport_autocert/) - WebTransport with AutoTLS (has go.mod)

### `third_party/` - Third-Party Integrations

Complete examples with their own `go.mod` files, demonstrating integration with popular Go libraries.

- [**`huma/`**](third_party/huma/) - Huma OpenAPI framework integration
- [**`pongo2/`**](third_party/pongo2/) - Django-style template engine
- [**`scs/`**](third_party/scs/) - Session management with SCS
- [**`templ/`**](third_party/templ/) - Templ HTML templating
- [**`zerolog/`**](third_party/zerolog/) - Structured logging with zerolog

### Other Examples

- [**`healthcheck/`**](healthcheck/) - Health check endpoints
- [**`metrics/`**](metrics/) - Prometheus metrics endpoint
- [**`pagination/`**](pagination/) - Pagination with response headers
- [**`pprof/`**](pprof/) - Performance profiling endpoints
- [**`sse/`**](sse/) - Server-Sent Events

## Running Examples

### Examples without external dependencies

Most core examples can be run directly:

```bash
cd core/hello_world
go run .
```

### Examples with third-party dependencies

Examples that require external dependencies have their own `go.mod` files. These are located in `third_party/` and some in `extensions/` or `middleware/` subdirectories.

```bash
cd third_party/zerolog
go mod tidy
go run .
```

For middleware examples with external dependencies (like `jwt_auth_golang_jwt/`), they have their own `go.mod`:

```bash
cd middleware/jwt_auth_golang_jwt
go mod tidy
go run .
```

## Common Patterns

### Basic Server
```go
app := zerohttp.New()
app.GET("/", handler)
log.Fatal(app.Start())
```

### With Configuration
```go
app := zerohttp.New(zh.Config{
    Addr: ":3000",
    TLS: zh.TLSConfig{
        Addr: ":3443",
    },
})
```

### Adding Middleware
```go
app.Use(compress.New())
app.Use(requestid.New())
```

See individual example directories for complete, runnable code.

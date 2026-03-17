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
- [**`healthcheck/`**](core/healthcheck/) - Health check endpoints
- [**`hello_world/`**](core/hello_world/) - Simplest possible server
- [**`lifecycle/`**](core/lifecycle/) - Server lifecycle hooks
- [**`metrics/`**](core/metrics/) - Prometheus metrics endpoint
- [**`pprof/`**](core/pprof/) - Performance profiling endpoints
- [**`problem_detail/`**](core/problem_detail/) - RFC 7807 Problem Detail responses
- [**`rendering/`**](core/rendering/) - Response rendering methods (JSON, HTML, text, blob, stream)
- [**`request_tracing/`**](core/request_tracing/) - Request ID propagation for tracing
- [**`route_groups/`**](core/route_groups/) - Route groups with nested middleware
- [**`sse/`**](extensions/sse/) - Server-Sent Events
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

- [**`basic_auth/`**](middleware/basic_auth/) - Basic authentication
- [**`cache/`**](middleware/cache/) - HTTP caching middleware
- [**`circuit_breaker/`**](middleware/circuit_breaker/) - Circuit breaker pattern
- [**`compress/`**](middleware/compress/) - Compression middleware (gzip/deflate)
- [**`compress_brotli/`**](middleware/compress_brotli/) - Brotli compression (has go.mod)
- [**`compress_zstd/`**](middleware/compress_zstd/) - Zstd compression (has go.mod)
- [**`content_charset/`**](middleware/content_charset/) - Content charset validation
- [**`content_encoding/`**](middleware/content_encoding/) - Content encoding validation
- [**`content_type/`**](middleware/content_type/) - Content type validation/middleware
- [**`cors/`**](middleware/cors/) - CORS handling
- [**`csrf/`**](middleware/csrf/) - CSRF protection
- [**`etag/`**](middleware/etag/) - ETag generation
- [**`hmac_auth/`**](middleware/hmac_auth/) - HMAC request signing
- [**`host_validation/`**](middleware/host_validation/) - Host header validation
- [**`idempotency/`**](middleware/idempotency/) - Idempotent request handling
- [**`jwt_auth/`**](middleware/jwt_auth/) - JWT authentication
- [**`jwt_auth_golang_jwt/`**](middleware/jwt_auth_golang_jwt/) - golang-jwt integration (has go.mod)
- [**`jwt_auth_lestrrat_jwx/`**](middleware/jwt_auth_lestrrat_jwx/) - lestrrat-go/jwx integration (has go.mod)
- [**`jwt_auth_refresh/`**](middleware/jwt_auth_refresh/) - JWT refresh token flow (has go.mod)
- [**`no_cache/`**](middleware/no_cache/) - Cache control headers
- [**`rate_limit/`**](middleware/rate_limit/) - Rate limiting (in-memory)
- [**`rate_limit_redis/`**](middleware/rate_limit_redis/) - Redis-backed rate limiting (has go.mod)
- [**`real_ip/`**](middleware/real_ip/) - Real IP extraction
- [**`recover/`**](middleware/recover/) - Panic recovery
- [**`request_body_size/`**](middleware/request_body_size/) - Request body size limits
- [**`request_id/`**](middleware/request_id/) - Request ID generation
- [**`request_logger/`**](middleware/request_logger/) - Request logging
- [**`reverse_proxy/`**](middleware/reverse_proxy/) - Reverse proxy setup
- [**`security_headers/`**](middleware/security_headers/) - Security headers
- [**`security_headers_nonce/`**](middleware/security_headers_nonce/) - CSP with nonces
- [**`set_header/`**](middleware/set_header/) - Header manipulation
- [**`timeout/`**](middleware/timeout/) - Request timeouts
- [**`tracing/`**](middleware/tracing/) - Distributed tracing
- [**`tracing_jaeger/`**](middleware/tracing_jaeger/) - Jaeger tracing (has go.mod)
- [**`tracing_otel/`**](middleware/tracing_otel/) - OpenTelemetry tracing (has go.mod)
- [**`trailing_slash/`**](middleware/trailing_slash/) - Trailing slash handling
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
- [**`zerolog/`**](third_party/zerolog/) - Structured logging with Zerolog

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
app := zerohttp.New(config.Config{
    Addr: ":3000",
    TLS: config.TLSConfig{
        Addr: ":3443",
    },
})
```

### Adding Middleware
```go
app.Use(middleware.Compress())
app.Use(middleware.RequestID())
```

See individual example directories for complete, runnable code.

// Package middleware provides HTTP middleware for zerohttp.
//
// Middleware can be applied at application, group, or route level:
//
//	// Application-level (all routes)
//	app.Use(middleware.CORS(config.DefaultCORSConfig))
//	app.Use(middleware.RequestID(config.RequestIDConfig{}))
//
//	// Route group
//	app.Group(func(api zh.Router) {
//	    api.Use(middleware.BasicAuth(config.BasicAuthConfig{
//	        Credentials: map[string]string{"admin": "secret"},
//	    }))
//	    api.GET("/admin", adminHandler)
//	})
//
//	// Route-specific
//	app.GET("/admin", adminHandler,
//	    middleware.BasicAuth(config.BasicAuthConfig{...}),
//	)
//
// # Available Middleware
//
// Authentication:
//   - [BasicAuth] - HTTP Basic Authentication
//   - [JWTAuth] - JWT token authentication with pluggable TokenStore
//   - [HMACAuth] - HMAC request signing (AWS Signature v4 style)
//
// Security:
//   - [CORS] - Cross-Origin Resource Sharing
//   - [CSRF] - Cross-Site Request Forgery protection
//   - [SecurityHeaders] - Security headers (CSP, HSTS, X-Frame-Options, etc.)
//   - [RequestBodySize] - Request body size limiting
//   - [HostValidation] - Host header validation
//
// Traffic Management:
//   - [RateLimit] - Token bucket or sliding window rate limiting
//   - [CircuitBreaker] - Circuit breaker pattern for fault tolerance
//   - [Timeout] - Request timeout handling
//   - [ReverseProxy] - Reverse proxy with load balancing
//
// Observability:
//   - [RequestLogger] - HTTP request/response logging
//   - [RequestID] - Request ID generation and propagation
//   - [RealIP] - Client IP extraction from proxy headers
//   - [Tracer] - Distributed tracing support
//
// Content:
//   - [Compress] - Gzip compression with configurable levels
//   - [ContentType] - Content-Type header enforcement
//   - [ContentEncoding] - Content-Encoding negotiation
//   - [ContentCharset] - Content-Type charset validation
//
// Caching:
//   - [Cache] - HTTP caching with ETag and Last-Modified
//   - [ETag] - ETag generation and conditional request handling
//   - [NoCache] - Cache-Control headers for dynamic content
//
// Utilities:
//   - [Recover] - Panic recovery middleware
//   - [TrailingSlash] - Trailing slash normalization
//   - [SetHeader] - Custom response header injection
//   - [Idempotency] - Idempotent request handling
//   - [WithValue] - Context value injection
//
// # Creating Custom Middleware
//
// Middleware follows the standard Go http.Handler pattern:
//
//	func MyMiddleware() func(http.Handler) http.Handler {
//	    return func(next http.Handler) http.Handler {
//	        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	            // Before handler
//	            start := time.Now()
//
//	            next.ServeHTTP(w, r)
//
//	            // After handler
//	            log.Printf("Request took %v", time.Since(start))
//	        })
//	    }
//	}
//
// # Middleware Execution Order
//
// Middleware executes in the order added:
//
//	app.Use(middleware.A) // First
//	app.Use(middleware.B) // Second
//
// Execution order: A → B → Handler → B (after) → A (after)
package middleware

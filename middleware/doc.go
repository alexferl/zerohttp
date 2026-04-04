// Package middleware provides HTTP middleware for zerohttp.
//
// Each middleware is in its own subpackage. Import the specific middleware you need:
//
//	import "github.com/alexferl/zerohttp/middleware/cors"
//	import "github.com/alexferl/zerohttp/middleware/basicauth"
//
// Middleware can be applied at application, group, or route level:
//
//	// Application-level (all routes)
//	app.Use(cors.New(cors.DefaultConfig))
//	app.Use(requestid.New())
//
//	// Route group
//	app.Group(func(api zh.Router) {
//	    api.Use(basicauth.New(basicauth.Config{
//	        Credentials: map[string]string{"admin": "secret"},
//	    }))
//	    api.GET("/admin", adminHandler)
//	})
//
//	// Route-specific
//	app.GET("/admin", adminHandler,
//	    basicauth.New(basicauth.Config{...}),
//	)
//
// # Available Middleware
//
// All middleware are in subpackages under middleware/:
//
// Authentication:
//   - [github.com/alexferl/zerohttp/middleware/basicauth] - HTTP Basic Authentication
//   - [github.com/alexferl/zerohttp/middleware/jwtauth] - JWT token authentication with pluggable TokenStore
//   - [github.com/alexferl/zerohttp/middleware/hmacauth] - HMAC request signing (AWS Signature v4 style)
//
// Security:
//   - [github.com/alexferl/zerohttp/middleware/cors] - Cross-Origin Resource Sharing
//   - [github.com/alexferl/zerohttp/middleware/csrf] - Cross-Site Request Forgery protection
//   - [github.com/alexferl/zerohttp/middleware/securityheaders] - Security headers (CSP, HSTS, X-Frame-Options, etc.)
//   - [github.com/alexferl/zerohttp/middleware/requestbodysize] - Request body size limiting
//   - [github.com/alexferl/zerohttp/middleware/host] - Host header validation
//
// Traffic Management:
//   - [github.com/alexferl/zerohttp/middleware/ratelimit] - Token bucket or sliding window rate limiting
//   - [github.com/alexferl/zerohttp/middleware/circuitbreaker] - Circuit breaker pattern for fault tolerance
//   - [github.com/alexferl/zerohttp/middleware/timeout] - Request timeout handling
//   - [github.com/alexferl/zerohttp/middleware/reverseproxy] - Reverse proxy with load balancing
//
// Observability:
//   - [github.com/alexferl/zerohttp/middleware/requestlogger] - HTTP request/response logging
//   - [github.com/alexferl/zerohttp/middleware/requestid] - Request ID generation and propagation
//   - [github.com/alexferl/zerohttp/middleware/realip] - Client IP extraction from proxy headers
//   - [github.com/alexferl/zerohttp/middleware/tracer] - Distributed tracing support
//
// Content:
//   - [github.com/alexferl/zerohttp/middleware/compress] - Gzip/Brotli/Zstd compression with configurable levels
//   - [github.com/alexferl/zerohttp/middleware/contenttype] - Content-Type header enforcement
//   - [github.com/alexferl/zerohttp/middleware/contentencoding] - Content-Encoding negotiation
//   - [github.com/alexferl/zerohttp/middleware/contentcharset] - Content-Type charset validation
//   - [github.com/alexferl/zerohttp/middleware/mediatype] - Media type negotiation with wildcard and suffix support
//
// Caching:
//   - [github.com/alexferl/zerohttp/middleware/cache] - HTTP caching with ETag and Last-Modified
//   - [github.com/alexferl/zerohttp/middleware/etag] - ETag generation and conditional request handling
//   - [github.com/alexferl/zerohttp/middleware/nocache] - Cache-Control headers for dynamic content
//
// Utilities:
//   - [github.com/alexferl/zerohttp/middleware/recover] - Panic recovery middleware
//   - [github.com/alexferl/zerohttp/middleware/trailingslash] - Trailing slash normalization
//   - [github.com/alexferl/zerohttp/middleware/setheader] - Custom response header injection
//   - [github.com/alexferl/zerohttp/middleware/idempotency] - Idempotent request handling
//   - [github.com/alexferl/zerohttp/middleware/value] - Context value injection
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
//	app.Use(middlewareA) // First
//	app.Use(middlewareB) // Second
//
// Execution order: A → B → Handler → B (after) → A (after)
package middleware

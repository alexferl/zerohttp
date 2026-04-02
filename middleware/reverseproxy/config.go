package reverseproxy

import (
	"net/http"
	"time"
)

// LoadBalancerAlgorithm defines the load balancing strategy
type LoadBalancerAlgorithm string

const (
	// RoundRobin cycles through backends in order
	RoundRobin LoadBalancerAlgorithm = "round_robin"
	// Random picks a random backend
	Random LoadBalancerAlgorithm = "random"
	// LeastConnections routes to backend with fewest active connections
	LeastConnections LoadBalancerAlgorithm = "least_connections"
)

// Backend represents a single upstream server
type Backend struct {
	// Target is the upstream URL (e.g., "http://localhost:8081")
	Target string

	// Weight is used for weighted load balancing.
	// Default: 1
	Weight int

	// Healthy indicates if the backend is currently healthy.
	// Default: true
	Healthy bool
}

// RewriteRule defines a path rewrite pattern
type RewriteRule struct {
	// Pattern is a glob pattern to match (e.g., "/api/v1/*")
	Pattern string

	// Replacement is the replacement path (e.g., "/api/v2/$1")
	// Use $1, $2, etc. to reference glob captures
	Replacement string
}

// Config allows customization of reverse proxy behavior
type Config struct {
	// Target is a single upstream URL (use this OR Targets, not both).
	// Example: "http://localhost:8081"
	// Default: ""
	Target string

	// Targets is a list of backends for load balancing (use this OR Target, not both).
	// Default: []
	Targets []Backend

	// LoadBalancer specifies the algorithm for multiple targets.
	// Default: RoundRobin
	LoadBalancer LoadBalancerAlgorithm

	// HealthCheckInterval is how often to check backend health (0 = disabled).
	// Default: 0 (disabled)
	HealthCheckInterval time.Duration

	// HealthCheckTimeout is the timeout for health check requests.
	// Default: 0 (uses http.DefaultTransport timeout)
	HealthCheckTimeout time.Duration

	// HealthCheckPath is the path to use for health checks.
	// Default: "/"
	HealthCheckPath string

	// StripPrefix removes this prefix from the request path before proxying.
	// Example: "/api/v1" -> request to "/api/v1/users" becomes "/users".
	// Default: "" (no stripping)
	StripPrefix string

	// AddPrefix adds this prefix to the request path after stripping.
	// Example: "/v2" -> request to "/users" becomes "/v2/users".
	// Default: "" (no prefix added)
	AddPrefix string

	// Rewrites is a list of path rewrite rules.
	// Example: [{Pattern: "/old/*", Replacement: "/new/$1"}].
	// Default: []
	Rewrites []RewriteRule

	// SetHeaders are headers to add/set on the outgoing request.
	// Default: {}
	SetHeaders map[string]string

	// RemoveHeaders are header names to remove from the outgoing request.
	// Default: []
	RemoveHeaders []string

	// ForwardHeaders automatically adds X-Forwarded-* headers.
	// X-Forwarded-For: Client IP
	// X-Forwarded-Proto: Original protocol (http/https)
	// X-Forwarded-Host: Original host header.
	// Default: true
	ForwardHeaders bool

	// ErrorHandler is called when the proxy encounters an error.
	// If nil, a default error handler is used.
	// Default: nil (uses default error handler)
	ErrorHandler func(http.ResponseWriter, *http.Request, error)

	// FallbackHandler is called when all backends are unavailable.
	// If nil, the ErrorHandler is used with a "Service Unavailable" error.
	// Default: nil
	FallbackHandler http.Handler

	// Transport allows customizing the HTTP transport.
	// If nil, http.DefaultTransport is used.
	// Default: nil (uses http.DefaultTransport)
	Transport http.RoundTripper

	// FlushInterval specifies the flush interval for streaming responses.
	// If 0, no periodic flushing.
	// Default: 0
	FlushInterval time.Duration

	// ModifyRequest allows customizing the outgoing request.
	// Called after all other modifications (strip/add prefix, headers, etc.).
	// Default: nil
	ModifyRequest func(*http.Request)

	// ModifyResponse allows customizing the response before it's written.
	// Return an error to trigger the error handler.
	// Default: nil
	ModifyResponse func(*http.Response) error

	// ExcludedPaths contains paths that skip reverse proxying.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where reverse proxying is explicitly applied.
	// If set, reverse proxying will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, reverse proxying applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains sensible defaults
var DefaultConfig = Config{
	LoadBalancer:    RoundRobin,
	HealthCheckPath: "/",
	StripPrefix:     "",
	AddPrefix:       "",
	Rewrites:        []RewriteRule{},
	SetHeaders:      map[string]string{},
	RemoveHeaders:   []string{},
	ForwardHeaders:  true,
	ExcludedPaths:   []string{},
	IncludedPaths:   []string{},
}

package config

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
	// Weight is used for weighted load balancing (defaults to 1)
	Weight int
	// Healthy indicates if the backend is currently healthy
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

// ReverseProxyConfig allows customization of reverse proxy behavior
type ReverseProxyConfig struct {
	// Target is a single upstream URL (use this OR Targets, not both)
	// Example: "http://localhost:8081"
	Target string

	// Targets is a list of backends for load balancing (use this OR Target, not both)
	Targets []Backend

	// LoadBalancer specifies the algorithm for multiple targets (defaults to RoundRobin)
	LoadBalancer LoadBalancerAlgorithm

	// HealthCheckInterval is how often to check backend health (0 = disabled)
	HealthCheckInterval time.Duration

	// HealthCheckTimeout is the timeout for health check requests
	HealthCheckTimeout time.Duration

	// HealthCheckPath is the path to use for health checks (defaults to "/")
	HealthCheckPath string

	// StripPrefix removes this prefix from the request path before proxying
	// Example: "/api/v1" -> request to "/api/v1/users" becomes "/users"
	StripPrefix string

	// AddPrefix adds this prefix to the request path after stripping
	// Example: "/v2" -> request to "/users" becomes "/v2/users"
	AddPrefix string

	// Rewrites is a list of path rewrite rules
	// Example: [{Pattern: "/old/*", Replacement: "/new/$1"}]
	Rewrites []RewriteRule

	// SetHeaders are headers to add/set on the outgoing request
	SetHeaders map[string]string

	// RemoveHeaders are header names to remove from the outgoing request
	RemoveHeaders []string

	// ForwardHeaders automatically adds X-Forwarded-* headers
	// X-Forwarded-For: Client IP
	// X-Forwarded-Proto: Original protocol (http/https)
	// X-Forwarded-Host: Original host header
	ForwardHeaders bool

	// ErrorHandler is called when the proxy encounters an error
	// If nil, a default error handler is used
	ErrorHandler func(http.ResponseWriter, *http.Request, error)

	// FallbackHandler is called when all backends are unavailable
	// If nil, the ErrorHandler is used with a "Service Unavailable" error
	FallbackHandler http.Handler

	// Transport allows customizing the HTTP transport
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper

	// FlushInterval specifies the flush interval for streaming responses
	// If 0, the default is used (no periodic flushing)
	FlushInterval time.Duration

	// ModifyRequest allows customizing the outgoing request
	// Called after all other modifications (strip/add prefix, headers, etc.)
	ModifyRequest func(*http.Request)

	// ModifyResponse allows customizing the response before it's written
	// Return an error to trigger the error handler
	ModifyResponse func(*http.Response) error

	// ExemptPaths contains paths that skip reverse proxying
	ExemptPaths []string
}

// DefaultReverseProxyConfig contains sensible defaults
var DefaultReverseProxyConfig = ReverseProxyConfig{
	LoadBalancer:    RoundRobin,
	HealthCheckPath: "/",
	StripPrefix:     "",
	AddPrefix:       "",
	Rewrites:        []RewriteRule{},
	SetHeaders:      map[string]string{},
	RemoveHeaders:   []string{},
	ForwardHeaders:  true,
	ExemptPaths:     []string{},
}

package circuitbreaker

import (
	"net/http"
	"time"
)

// Config allows customization of circuit breaker behavior
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	// Default: 5
	FailureThreshold int

	// RecoveryTimeout is how long to wait before trying to close the circuit.
	// Default: 30s
	RecoveryTimeout time.Duration

	// SuccessThreshold is the number of consecutive successes needed to close the circuit from half-open.
	// Default: 3
	SuccessThreshold int

	// MaxHalfOpenRequests is the maximum number of concurrent requests allowed in half-open state.
	// This prevents thundering herd when service recovers.
	// Default: 1
	MaxHalfOpenRequests int

	// IsFailure determines if a response should be considered a failure.
	// Default: 5xx status codes are considered failures
	IsFailure func(*http.Request, int) bool

	// KeyExtractor extracts the circuit breaker key from the request (for per-endpoint circuits).
	// Default: r.URL.Path (per-endpoint circuit breaker)
	KeyExtractor func(*http.Request) string

	// OpenStatusCode is the status code to return when circuit is open.
	// Default: 503 (Service Unavailable)
	OpenStatusCode int

	// OpenMessage is the message to return when circuit is open.
	// Default: "Service temporarily unavailable"
	OpenMessage string
}

// DefaultConfig contains the default values for circuit breaker configuration.
var DefaultConfig = Config{
	FailureThreshold:    5,
	RecoveryTimeout:     30 * time.Second,
	SuccessThreshold:    3,
	MaxHalfOpenRequests: 1,
	IsFailure: func(r *http.Request, statusCode int) bool {
		return statusCode >= http.StatusInternalServerError // Consider 5xx as failures
	},
	KeyExtractor: func(r *http.Request) string {
		return r.URL.Path // Per-endpoint circuit breaker
	},
	OpenStatusCode: http.StatusServiceUnavailable,
	OpenMessage:    "Service temporarily unavailable",
}

package config

import (
	"net/http"
	"time"
)

// CircuitBreakerConfig allows customization of circuit breaker behavior
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int
	// RecoveryTimeout is how long to wait before trying to close the circuit
	RecoveryTimeout time.Duration
	// SuccessThreshold is the number of consecutive successes needed to close the circuit from half-open
	SuccessThreshold int
	// IsFailure determines if a response should be considered a failure
	IsFailure func(*http.Request, int) bool
	// KeyExtractor extracts the circuit breaker key from the request (for per-endpoint circuits)
	KeyExtractor func(*http.Request) string
	// OpenStatusCode is the status code to return when circuit is open
	OpenStatusCode int
	// OpenMessage is the message to return when circuit is open
	OpenMessage string
}

// DefaultCircuitBreakerConfig contains the default values for circuit breaker configuration.
var DefaultCircuitBreakerConfig = CircuitBreakerConfig{
	FailureThreshold: 5,
	RecoveryTimeout:  30 * time.Second,
	SuccessThreshold: 3,
	IsFailure: func(r *http.Request, statusCode int) bool {
		return statusCode >= http.StatusInternalServerError // Consider 5xx as failures
	},
	KeyExtractor: func(r *http.Request) string {
		return r.URL.Path // Per-endpoint circuit breaker
	},
	OpenStatusCode: http.StatusServiceUnavailable,
	OpenMessage:    "Service temporarily unavailable",
}

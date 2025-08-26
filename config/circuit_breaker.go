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
		return statusCode >= 500 // Consider 5xx as failures
	},
	KeyExtractor: func(r *http.Request) string {
		return r.URL.Path // Per-endpoint circuit breaker
	},
	OpenStatusCode: http.StatusServiceUnavailable,
	OpenMessage:    "Service temporarily unavailable",
}

// CircuitBreakerOption configures circuit breaker middleware.
type CircuitBreakerOption func(*CircuitBreakerConfig)

// WithCircuitBreakerFailureThreshold sets the number of consecutive failures before opening the circuit.
func WithCircuitBreakerFailureThreshold(threshold int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.FailureThreshold = threshold
	}
}

// WithCircuitBreakerRecoveryTimeout sets how long to wait before trying to close the circuit.
func WithCircuitBreakerRecoveryTimeout(timeout time.Duration) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.RecoveryTimeout = timeout
	}
}

// WithCircuitBreakerSuccessThreshold sets the number of consecutive successes needed to close the circuit from half-open.
func WithCircuitBreakerSuccessThreshold(threshold int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.SuccessThreshold = threshold
	}
}

// WithCircuitBreakerIsFailure sets the function that determines if a response should be considered a failure.
func WithCircuitBreakerIsFailure(isFailure func(*http.Request, int) bool) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.IsFailure = isFailure
	}
}

// WithCircuitBreakerKeyExtractor sets the function that extracts the circuit breaker key from the request.
func WithCircuitBreakerKeyExtractor(keyExtractor func(*http.Request) string) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.KeyExtractor = keyExtractor
	}
}

// WithCircuitBreakerOpenStatusCode sets the status code to return when circuit is open.
func WithCircuitBreakerOpenStatusCode(statusCode int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.OpenStatusCode = statusCode
	}
}

// WithCircuitBreakerOpenMessage sets the message to return when circuit is open.
func WithCircuitBreakerOpenMessage(message string) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.OpenMessage = message
	}
}

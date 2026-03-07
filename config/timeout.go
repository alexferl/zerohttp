package config

import (
	"net/http"
	"time"
)

// TimeoutConfig allows customization of request timeout behavior
type TimeoutConfig struct {
	// Timeout duration for the request (defaults to 30 seconds)
	Timeout time.Duration
	// StatusCode to return on timeout (defaults to 504 Gateway Timeout)
	StatusCode int
	// Message to write on timeout (optional)
	Message string
	// ExemptPaths contains paths that skip timeout enforcement
	ExemptPaths []string
}

// DefaultTimeoutConfig contains the default values for timeout configuration.
var DefaultTimeoutConfig = TimeoutConfig{
	Timeout:     30 * time.Second,
	StatusCode:  http.StatusGatewayTimeout,
	Message:     "",
	ExemptPaths: []string{},
}

// TimeoutOption configures timeout middleware.
type TimeoutOption func(*TimeoutConfig)

// WithTimeoutDuration sets the timeout duration for requests.
func WithTimeoutDuration(timeout time.Duration) TimeoutOption {
	return func(c *TimeoutConfig) {
		c.Timeout = timeout
	}
}

// WithTimeoutStatusCode sets the status code to return on timeout.
func WithTimeoutStatusCode(statusCode int) TimeoutOption {
	return func(c *TimeoutConfig) {
		c.StatusCode = statusCode
	}
}

// WithTimeoutMessage sets the message to write on timeout.
func WithTimeoutMessage(message string) TimeoutOption {
	return func(c *TimeoutConfig) {
		c.Message = message
	}
}

// WithTimeoutExemptPaths sets paths that skip timeout enforcement.
func WithTimeoutExemptPaths(paths []string) TimeoutOption {
	return func(c *TimeoutConfig) {
		c.ExemptPaths = paths
	}
}

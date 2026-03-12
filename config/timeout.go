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

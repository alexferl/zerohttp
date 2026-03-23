package timeout

import (
	"net/http"
	"time"
)

// Config allows customization of request timeout behavior
type Config struct {
	// Timeout duration for the request (defaults to 30 seconds)
	Timeout time.Duration

	// StatusCode to return on timeout (defaults to 504 Gateway Timeout)
	StatusCode int

	// Message to write on timeout (optional)
	Message string

	// ExcludedPaths contains paths that skip timeout enforcement.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where timeout is explicitly applied.
	// If set, timeout will only be enforced for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, timeout applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default values for timeout configuration.
var DefaultConfig = Config{
	Timeout:       30 * time.Second,
	StatusCode:    http.StatusGatewayTimeout,
	Message:       "",
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
}

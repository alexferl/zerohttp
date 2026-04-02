package etag

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
)

// Algorithm defines the hashing algorithm for ETag generation
type Algorithm string

const (
	// FNV uses FNV-1a 64-bit hash (fast, good for most use cases)
	FNV Algorithm = "fnv"
	// MD5 uses MD5 hash (slower, more collision-resistant)
	MD5 Algorithm = "md5"
)

// Config allows customization of ETag middleware behavior
type Config struct {
	// Algorithm selects the hashing function.
	// Default: FNV
	Algorithm Algorithm

	// Weak determines if ETags should be prefixed with "W/".
	// Use a pointer to distinguish between "not set" and "explicitly set to false".
	// Default: false
	Weak *bool

	// MaxBufferSize is the maximum response body size to buffer for ETag generation.
	// Default: 1MB
	MaxBufferSize int64

	// SkipStatusCodes contains status codes that should not have ETags generated.
	// Default: error status codes (4xx, 5xx, redirects)
	SkipStatusCodes map[int]struct{}

	// SkipContentTypes contains content types that should not have ETags generated.
	// Default: [text/event-stream]
	SkipContentTypes map[string]struct{}

	// ExcludedPaths contains paths to skip ETag generation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where ETag generation is explicitly applied.
	// If set, ETag will only be generated for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, ETag applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// ExcludedFunc is a custom function to determine if ETag generation should be skipped for a request
	ExcludedFunc func(r *http.Request) bool
}

// DefaultConfig contains the default values for ETag configuration
var DefaultConfig = Config{
	Algorithm:     FNV,
	Weak:          config.Bool(false),
	MaxBufferSize: 1024 * 1024, // 1MB
	SkipStatusCodes: map[int]struct{}{
		http.StatusNoContent:           {},
		http.StatusPartialContent:      {},
		http.StatusMovedPermanently:    {},
		http.StatusFound:               {},
		http.StatusNotModified:         {},
		http.StatusTemporaryRedirect:   {},
		http.StatusPermanentRedirect:   {},
		http.StatusBadRequest:          {},
		http.StatusUnauthorized:        {},
		http.StatusForbidden:           {},
		http.StatusNotFound:            {},
		http.StatusMethodNotAllowed:    {},
		http.StatusInternalServerError: {},
		http.StatusBadGateway:          {},
		http.StatusServiceUnavailable:  {},
	},
	SkipContentTypes: map[string]struct{}{
		httpx.MIMETextEventStream: {}, // SSE streaming
	},
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
	ExcludedFunc:  nil,
}

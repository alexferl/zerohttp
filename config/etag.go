package config

import "net/http"

// ETagAlgorithm defines the hashing algorithm for ETag generation
type ETagAlgorithm string

const (
	// FNV uses FNV-1a 64-bit hash (fast, good for most use cases)
	FNV ETagAlgorithm = "fnv"
	// MD5 uses MD5 hash (slower, more collision-resistant)
	MD5 ETagAlgorithm = "md5"
)

// ETagConfig allows customization of ETag middleware behavior
type ETagConfig struct {
	// Algorithm selects the hashing function (defaults to FNV)
	Algorithm ETagAlgorithm

	// Weak determines if ETags should be prefixed with "W/" (defaults to true)
	// Use a pointer to distinguish between "not set" and "explicitly set to false"
	Weak *bool

	// MaxBufferSize is the maximum response body size to buffer for ETag generation (defaults to 1MB)
	MaxBufferSize int

	// SkipStatusCodes contains status codes that should not have ETags generated (defaults to error status codes)
	SkipStatusCodes map[int]struct{}

	// SkipContentTypes contains content types that should not have ETags generated
	SkipContentTypes map[string]struct{}

	// ExemptPaths contains paths to skip ETag generation
	ExemptPaths []string

	// ExemptFunc is a custom function to determine if ETag generation should be skipped for a request
	ExemptFunc func(r *http.Request) bool
}

// DefaultETagConfig contains the default values for ETag configuration
var DefaultETagConfig = ETagConfig{
	Algorithm:     FNV,
	Weak:          Bool(true),
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
		"text/event-stream": {}, // SSE streaming
	},
	ExemptPaths: []string{},
	ExemptFunc:  nil,
}

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
	Weak bool
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
	Weak:          true,
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

// ETagOption configures ETag middleware
type ETagOption func(*ETagConfig)

// WithETagAlgorithm sets the hashing algorithm for ETag generation
func WithETagAlgorithm(algorithm ETagAlgorithm) ETagOption {
	return func(c *ETagConfig) {
		c.Algorithm = algorithm
	}
}

// WithETagWeak sets whether ETags should be weak (prefixed with "W/")
func WithETagWeak(weak bool) ETagOption {
	return func(c *ETagConfig) {
		c.Weak = weak
	}
}

// WithETagMaxBufferSize sets the maximum response body size to buffer for ETag generation
func WithETagMaxBufferSize(size int) ETagOption {
	return func(c *ETagConfig) {
		c.MaxBufferSize = size
	}
}

// WithETagSkipStatusCodes sets status codes that should not have ETags generated
func WithETagSkipStatusCodes(codes ...int) ETagOption {
	return func(c *ETagConfig) {
		c.SkipStatusCodes = make(map[int]struct{})
		for _, code := range codes {
			c.SkipStatusCodes[code] = struct{}{}
		}
	}
}

// WithETagSkipContentTypes sets content types that should not have ETags generated
func WithETagSkipContentTypes(types ...string) ETagOption {
	return func(c *ETagConfig) {
		c.SkipContentTypes = make(map[string]struct{})
		for _, t := range types {
			c.SkipContentTypes[t] = struct{}{}
		}
	}
}

// WithETagExemptPaths sets paths to skip ETag generation
func WithETagExemptPaths(paths []string) ETagOption {
	return func(c *ETagConfig) {
		c.ExemptPaths = paths
	}
}

// WithETagExemptFunc sets a custom function to determine if ETag generation should be skipped
func WithETagExemptFunc(fn func(r *http.Request) bool) ETagOption {
	return func(c *ETagConfig) {
		c.ExemptFunc = fn
	}
}

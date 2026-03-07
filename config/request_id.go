package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// RequestIDContextKey is a custom type for context keys to avoid collisions.
type RequestIDContextKey string

// RequestIDConfig allows customization of request ID generation.
type RequestIDConfig struct {
	// Header is the header name for the request ID (defaults to "X-Request-Id").
	Header string
	// Generator is a custom function to generate request IDs.
	Generator func() string
	// ContextKey is the key to store the request ID in context (defaults to "request_id").
	ContextKey RequestIDContextKey
}

// DefaultRequestIDConfig contains the default configuration for request ID generation.
var DefaultRequestIDConfig = RequestIDConfig{
	Header:     "X-Request-Id",
	Generator:  GenerateRequestID,
	ContextKey: RequestIDContextKey("request_id"),
}

// GenerateRequestID creates a unique request ID.
func GenerateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("request-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// RequestIDOption configures request ID generation middleware.
type RequestIDOption func(*RequestIDConfig)

// WithRequestIDHeader sets the header name for the request ID.
func WithRequestIDHeader(header string) RequestIDOption {
	return func(c *RequestIDConfig) {
		c.Header = header
	}
}

// WithRequestIDGenerator sets a custom request ID generator function.
func WithRequestIDGenerator(gen func() string) RequestIDOption {
	return func(c *RequestIDConfig) {
		c.Generator = gen
	}
}

// WithRequestIDContextKey sets the context key used to store the request ID.
func WithRequestIDContextKey(key RequestIDContextKey) RequestIDOption {
	return func(c *RequestIDConfig) {
		c.ContextKey = key
	}
}

// requestIDConfigToOptions converts a RequestIDConfig struct to a slice of RequestIDOption functions.
func requestIDConfigToOptions(cfg RequestIDConfig) []RequestIDOption {
	return []RequestIDOption{
		WithRequestIDHeader(cfg.Header),
		WithRequestIDGenerator(cfg.Generator),
		WithRequestIDContextKey(cfg.ContextKey),
	}
}

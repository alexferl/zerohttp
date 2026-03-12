package config

import (
	"crypto/rand"
	"encoding/hex"
)

// RequestIDContextKey is a custom type for context keys to avoid collisions.
type RequestIDContextKey string

// RequestIDConfig allows customization of request ID generation.
type RequestIDConfig struct {
	// Header is the header name for the request ID (defaults to "X-Request-Id").
	Header string
	// Generator is a custom function to generate request IDs.
	// The default generator uses crypto/rand (CSPRNG) for 128 bits of entropy.
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

// GenerateRequestID creates a unique request ID using crypto/rand.
// Returns a 32-character hex string with 128 bits of entropy.
func GenerateRequestID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

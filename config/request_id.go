package config

import (
	"crypto/rand"
	"encoding/hex"
)

// requestIDContextKey is the context key type for request ID.
type requestIDContextKey struct{}

// RequestIDContextKey is the context key for request ID.
var RequestIDContextKey = requestIDContextKey{}

// RequestIDConfig allows customization of request ID generation.
type RequestIDConfig struct {
	// Header is the header name for the request ID (defaults to "X-Request-Id").
	Header string

	// Generator is a custom function to generate request IDs.
	// The default generator uses crypto/rand (CSPRNG) for 128 bits of entropy.
	Generator func() string

	// ContextKey is the key to store the request ID in context.
	// Defaults to the package-provided RequestIDContextKey.
	ContextKey any
}

// DefaultRequestIDConfig contains the default configuration for request ID generation.
var DefaultRequestIDConfig = RequestIDConfig{
	Header:     "X-Request-Id",
	Generator:  GenerateRequestID,
	ContextKey: RequestIDContextKey,
}

// GenerateRequestID creates a unique request ID using crypto/rand.
// Returns a 32-character hex string with 128 bits of entropy.
func GenerateRequestID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

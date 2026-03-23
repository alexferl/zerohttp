package requestid

import (
	"crypto/rand"
	"encoding/hex"
)

// contextKey is the context key type for request ID.
type contextKey struct{}

// ContextKey is the context key for request ID.
var ContextKey = contextKey{}

// Config allows customization of request ID generation.
type Config struct {
	// Header is the header name for the request ID (defaults to "X-Request-Id").
	Header string

	// Generator is a custom function to generate request IDs.
	// The default generator uses crypto/rand (CSPRNG) for 128 bits of entropy.
	Generator func() string

	// ContextKey is the key to store the request ID in context.
	// Defaults to the package-provided ContextKey.
	ContextKey any
}

// DefaultConfig contains the default configuration for request ID generation.
var DefaultConfig = Config{
	Header:     "X-Request-Id",
	Generator:  GenerateRequestID,
	ContextKey: ContextKey,
}

// GenerateRequestID creates a unique request ID using crypto/rand.
// Returns a 32-character hex string with 128 bits of entropy.
func GenerateRequestID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

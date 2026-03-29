package jwtauth

import (
	"context"
	"time"
)

// RevocationStore defines the interface for JWT token revocation storage.
// This is a subset of the Store interface, focused only on revocation operations.
type RevocationStore interface {
	// Revoke invalidates a token by its JTI (JWT ID).
	// The TTL should be set to the remaining token lifetime.
	// Returns an error if the revocation operation fails.
	Revoke(ctx context.Context, jti string, ttl time.Duration) error

	// IsRevoked checks if a token has been revoked by its JTI.
	// Returns (true, nil) if revoked, (false, nil) if not revoked.
	// Returns error if the check fails.
	IsRevoked(ctx context.Context, jti string) (bool, error)

	// Close releases resources associated with the store.
	// Returns an error if the close operation fails.
	Close() error
}

// GetJTI extracts the JWT ID (jti) claim from the claims map.
// Returns empty string if jti is not present or not a string.
func GetJTI(claims map[string]any) string {
	if claims == nil {
		return ""
	}
	if jti, ok := claims[JWTClaimJWTID].(string); ok {
		return jti
	}
	return ""
}

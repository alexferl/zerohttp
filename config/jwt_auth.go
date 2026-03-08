package config

import (
	"net/http"
	"strings"
	"time"
)

// TokenType indicates the type of token being generated
type TokenType int

const (
	AccessToken TokenType = iota
	RefreshToken
)

const (
	TokenTypeRefresh = "refresh"
)

// TokenStore is the interface for JWT token operations.
// Users implement this interface to integrate their preferred JWT library
// and handle token persistence (generation, validation, revocation).
type TokenStore interface {
	// Validate parses and validates a JWT token, returning the claims.
	// Returns (claims, nil) on valid token, (nil, error) on invalid.
	Validate(token string) (JWTClaims, error)

	// Generate creates a new signed JWT token for the given claims.
	// Used for access tokens and refresh tokens.
	Generate(claims JWTClaims, tokenType TokenType) (string, error)

	// Revoke invalidates a refresh token (called during logout).
	// Implement this to store revoked token identifiers (e.g., jti) in database/Redis.
	// Return nil if revocation succeeds or if token doesn't need revocation.
	Revoke(claims JWTClaims) error

	// IsRevoked checks if a refresh token has been revoked.
	// Return true if token was revoked, false otherwise.
	// Called during token refresh to prevent use of revoked tokens.
	IsRevoked(claims JWTClaims) bool
}

// JWTAuthConfig configures JWT authentication middleware
type JWTAuthConfig struct {
	// TokenExtractor extracts the JWT token from the request.
	// Default: extracts from "Authorization: Bearer <token>" header
	TokenExtractor func(r *http.Request) string

	// TokenStore handles all token operations (validate, generate, revoke).
	// This is the PLUGGABLE INTERFACE - users implement this with their JWT library.
	// REQUIRED for authentication to work.
	TokenStore TokenStore

	// RequiredClaims are claims that MUST be present in the token.
	// Validation fails if any are missing.
	// Default: none
	RequiredClaims []string

	// ExemptPaths are paths that skip JWT validation.
	// Default: []
	ExemptPaths []string

	// ExemptMethods are HTTP methods that skip JWT validation.
	// Default: [] (OPTIONS is always exempt)
	ExemptMethods []string

	// ErrorHandler is called when JWT validation fails.
	// Default: Returns 401/403 with RFC 9457 Problem Details
	ErrorHandler http.HandlerFunc

	// OnSuccess is called after successful validation (optional).
	// Use for audit logging, metrics, etc.
	OnSuccess func(r *http.Request, claims JWTClaims)

	// AccessTokenTTL is the time-to-live for access tokens.
	// Default: 15 minutes
	AccessTokenTTL time.Duration

	// RefreshTokenTTL is the time-to-live for refresh tokens.
	// Default: 7 days
	RefreshTokenTTL time.Duration
}

// DefaultJWTAuthConfig provides sensible defaults
var DefaultJWTAuthConfig = JWTAuthConfig{
	TokenExtractor:  extractBearerToken,
	ExemptPaths:     []string{},
	ExemptMethods:   []string{},
	RequiredClaims:  []string{},
	AccessTokenTTL:  15 * time.Minute,
	RefreshTokenTTL: 7 * 24 * time.Hour,
}

// JWTClaims represents validated JWT claims.
// This is intentionally an interface{} to allow any JWT library's claims type.
// Users type-assert to their library's claims type in handlers.
type JWTClaims any

// Standard JWT claim keys (RFC 7519)
const (
	JWTClaimSubject    = "sub"
	JWTClaimIssuer     = "iss"
	JWTClaimAudience   = "aud"
	JWTClaimExpiration = "exp"
	JWTClaimNotBefore  = "nbf"
	JWTClaimIssuedAt   = "iat"
	JWTClaimJWTID      = "jti"
	JWTClaimScope      = "scope"
	JWTClaimType       = "type"
)

// extractBearerToken extracts the JWT token from the Authorization header
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimSpace(auth[len(prefix):])
}

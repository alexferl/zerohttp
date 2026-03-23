package jwtauth

import (
	"context"
	"net/http"
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

// Store is the interface for JWT token operations.
// Users implement this interface to integrate their preferred JWT library
// and handle token persistence (generation, validation, revocation).
//
// Best Practice: Validate() should return map[string]any for maximum compatibility
// with the middleware. If you return a custom type, claims normalization will
// convert it using reflection, but map[string]any is fastest and most reliable.
type Store interface {
	// Validate parses and validates a JWT token, returning the claims.
	// Returns (claims, nil) on valid token, (nil, error) on invalid.
	//
	// RECOMMENDED: Return map[string]any for best performance and compatibility.
	// The middleware will normalize any returned type to map[string]any anyway.
	Validate(ctx context.Context, token string) (JWTClaims, error)

	// Generate creates a new signed JWT token for the given claims.
	// Used for access tokens and refresh tokens.
	//
	// Note: The TTL is provided so you can set the exp claim correctly for your
	// JWT library. Some libraries expect time.Time, others expect Unix timestamp.
	Generate(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error)

	// Revoke invalidates a refresh token (called during logout/refresh).
	// Implement this to store revoked token identifiers (e.g., jti) in database/Redis.
	// Return nil if revocation succeeds or if token doesn't need revocation.
	//
	// Note: claims is always passed as map[string]any for consistency.
	Revoke(ctx context.Context, claims map[string]any) error

	// IsRevoked checks if a refresh token has been revoked.
	// Return (true, nil) if token was revoked, (false, nil) if not revoked.
	// Return error if the check fails (e.g., database connection error).
	// Called during token refresh to prevent use of revoked tokens.
	//
	// Note: claims is always passed as map[string]any for consistency.
	IsRevoked(ctx context.Context, claims map[string]any) (bool, error)
}

// Config configures JWT authentication middleware
type Config struct {
	// Extractor extracts the JWT token from the request.
	// Default: extracts from "Authorization: Bearer <token>" header
	Extractor func(r *http.Request) string

	// Store handles all token operations (validate, generate, revoke).
	// This is the PLUGGABLE INTERFACE - users implement this with their JWT library.
	// REQUIRED for authentication to work.
	Store Store

	// RequiredClaims are claims that MUST be present in the token.
	// Validation fails if any are missing.
	// Default: none
	RequiredClaims []string

	// ExcludedPaths are paths that skip JWT validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where JWT validation is explicitly applied.
	// If set, JWT validation will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, JWT validation applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// ExcludedMethods are HTTP methods that skip JWT validation.
	// Default: [] (OPTIONS is always excluded)
	ExcludedMethods []string

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

// DefaultConfig provides sensible defaults
var DefaultConfig = Config{
	Extractor:       extractBearerToken,
	ExcludedPaths:   []string{},
	IncludedPaths:   []string{},
	ExcludedMethods: []string{},
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

package config

import (
	"net/http"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// HMACHashAlgorithm represents the supported HMAC hash algorithms
type HMACHashAlgorithm string

const (
	HMACSHA256 HMACHashAlgorithm = "SHA256"
	HMACSHA384 HMACHashAlgorithm = "SHA384"
	HMACSHA512 HMACHashAlgorithm = "SHA512"
)

// HMACAuthConfig configures HMAC request signing authentication
type HMACAuthConfig struct {
	// CredentialStore retrieves the secret key(s) for a given access key ID.
	// Returns a slice of valid secrets - multiple secrets enable key rotation.
	// During rotation, both old and new keys can be valid simultaneously.
	// Return nil or empty slice if access key ID is not found.
	// REQUIRED.
	CredentialStore func(accessKeyID string) []string

	// Algorithm is the HMAC hash algorithm to use.
	// Default: HMACSHA256
	Algorithm HMACHashAlgorithm

	// MaxSkew is the maximum allowed time difference between request timestamp
	// and server time. Used for replay attack protection.
	// Default: 5 minutes
	MaxSkew time.Duration

	// ClockSkewGrace adds extra tolerance for clock skew between client and server.
	// Default: 1 minute
	ClockSkewGrace time.Duration

	// RequiredHeaders are headers that must be present and included in signature.
	// Default: ["host", "x-timestamp"]
	RequiredHeaders []string

	// OptionalHeaders are headers that will be signed if present.
	// Default: ["content-type"]
	OptionalHeaders []string

	// ExcludedPaths are paths that skip HMAC validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where HMAC validation is explicitly applied.
	// If set, HMAC validation will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, HMAC validation applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// ErrorHandler is called when HMAC validation fails.
	// Default: Returns 401 Unauthorized with RFC 9457 Problem Details
	ErrorHandler http.HandlerFunc

	// AuthHeaderName is the header containing the authorization.
	// Default: "Authorization"
	AuthHeaderName string

	// TimestampHeader is the header containing the request timestamp.
	// Must be ISO8601 format (e.g., 2026-03-07T12:00:00Z).
	// Default: "X-Timestamp"
	TimestampHeader string

	// AllowUnsignedPayload allows requests without body signature for streaming.
	// When true, body hash is set to "UNSIGNED-PAYLOAD" and not verified.
	// Default: false
	AllowUnsignedPayload bool

	// AllowPresignedURLs allows authentication via query string parameters.
	// When true, the middleware will check for HMAC parameters in the URL query
	// (X-HMAC-Algorithm, X-HMAC-Credential, X-HMAC-SignedHeaders, X-HMAC-Signature)
	// as an alternative to the Authorization header. This is useful for pre-signed URLs.
	// Default: false
	AllowPresignedURLs bool

	// AuditLogger is called for every HMAC authentication attempt.
	// Use this for logging auth successes and failures to a security log.
	// Default: nil (no audit logging)
	AuditLogger HMACAuditLogger

	// MaxBodySize is the maximum body size in bytes that will be read for computing
	// the body hash. Requests with bodies larger than this will be rejected with
	// 413 Payload Too Large unless AllowUnsignedPayload is true.
	// Default: 10MB
	MaxBodySize int64
}

// HMACAuditLogger is called for each authentication attempt.
// The success parameter indicates whether authentication passed or failed.
// The errType parameter describes the type of failure (empty string on success).
type HMACAuditLogger func(accessKeyID string, timestamp time.Time, success bool, errType string)

// DefaultHMACAuthConfig provides sensible defaults
var DefaultHMACAuthConfig = HMACAuthConfig{
	CredentialStore:      nil, // Must be set by user
	Algorithm:            HMACSHA256,
	MaxSkew:              5 * time.Minute,
	ClockSkewGrace:       1 * time.Minute,
	RequiredHeaders:      []string{"host", "x-timestamp"},
	OptionalHeaders:      []string{"content-type"},
	ExcludedPaths:        []string{},
	IncludedPaths:        []string{},
	ErrorHandler:         nil,
	AuthHeaderName:       httpx.HeaderAuthorization,
	TimestampHeader:      httpx.HeaderXTimestamp,
	AllowUnsignedPayload: false,
	MaxBodySize:          10 * 1024 * 1024, // 10MB default
}

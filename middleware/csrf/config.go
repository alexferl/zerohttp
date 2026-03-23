package csrf

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// Config holds configuration for CSRF protection
type Config struct {
	// CookieName is the name of the CSRF cookie
	// Default: "csrf_token"
	CookieName string

	// CookieMaxAge is the max age of the CSRF cookie in seconds
	// Default: 86400 (24 hours)
	CookieMaxAge int

	// CookieDomain sets the domain for the CSRF cookie
	// Default: "" (current domain only)
	CookieDomain string

	// CookiePath sets the path for the CSRF cookie
	// Default: "/"
	CookiePath string

	// CookieSecure sets the Secure flag on the cookie
	// Default: true (recommended for HTTPS)
	// Use a pointer to distinguish between "not set" and "explicitly set to false"
	CookieSecure *bool

	// CookieSameSite sets the SameSite attribute
	// Default: http.SameSiteStrictMode
	CookieSameSite http.SameSite

	// TokenLookup allows extracting token from multiple sources
	// Format: "<source>:<name>" where source is "header", "form", or "query"
	// Default: "header:X-CSRF-Token"
	TokenLookup string

	// ErrorHandler is called when CSRF validation fails
	// Default: Returns 403 Forbidden with plain text error
	ErrorHandler http.HandlerFunc

	// ExcludedPaths contains paths that skip CSRF validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where CSRF validation is explicitly applied.
	// If set, CSRF will only be validated for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, CSRF applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// ExcludedMethods contains HTTP methods that skip CSRF validation
	// Default: []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	ExcludedMethods []string

	// HMACKey is the secret key used for HMAC signing (REQUIRED)
	// Must be set explicitly. The middleware will panic if not set.
	// Use a 32-byte key from environment variables or secure storage.
	// All servers in a cluster must use the same key.
	HMACKey []byte

	// TokenGenerator is an optional function for generating CSRF tokens.
	// Defaults to crypto-secure random generation.
	TokenGenerator func(key []byte) (string, error)
}

// DefaultConfig contains the default values for CSRF configuration
var DefaultConfig = Config{
	CookieName:      "csrf_token",
	CookieMaxAge:    86400, // 24 hours
	CookieDomain:    "",
	CookiePath:      "/",
	CookieSecure:    config.Bool(true),
	CookieSameSite:  http.SameSiteStrictMode,
	TokenLookup:     "header:X-CSRF-Token",
	ErrorHandler:    nil,
	ExcludedPaths:   []string{},
	IncludedPaths:   []string{},
	ExcludedMethods: []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace},
	HMACKey:         nil,
}

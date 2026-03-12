package config

import (
	"net/http"
)

// CSRFConfig holds configuration for CSRF protection
type CSRFConfig struct {
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

	// ExemptPaths contains paths that skip CSRF validation
	// Default: []string{}
	ExemptPaths []string

	// ExemptMethods contains HTTP methods that skip CSRF validation
	// Default: []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
	ExemptMethods []string

	// HMACKey is the secret key used for HMAC signing (REQUIRED)
	// Must be set explicitly. The middleware will panic if not set.
	// Use a 32-byte key from environment variables or secure storage.
	// All servers in a cluster must use the same key.
	HMACKey []byte

	// TokenGenerator is an optional function for generating CSRF tokens.
	// Defaults to crypto-secure random generation.
	TokenGenerator func(key []byte) (string, error)
}

// DefaultCSRFConfig contains the default values for CSRF configuration
var DefaultCSRFConfig = CSRFConfig{
	CookieName:     "csrf_token",
	CookieMaxAge:   86400, // 24 hours
	CookieDomain:   "",
	CookiePath:     "/",
	CookieSecure:   Bool(true),
	CookieSameSite: http.SameSiteStrictMode,
	TokenLookup:    "header:X-CSRF-Token",
	ErrorHandler:   nil,
	ExemptPaths:    []string{},
	ExemptMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace},
	HMACKey:        nil,
}

package config

import "net/http"

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
	CookieSecure bool

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
	// Default: []string{"GET", "HEAD", "OPTIONS", "TRACE"}
	ExemptMethods []string

	// HMACKey is the secret key used for HMAC signing (REQUIRED)
	// Must be set with WithCSRFHMACKey(). The middleware will panic if not set.
	// Use a 32-byte key from environment variables or secure storage.
	// All servers in a cluster must use the same key.
	HMACKey []byte
}

// DefaultCSRFConfig contains the default values for CSRF configuration
var DefaultCSRFConfig = CSRFConfig{
	CookieName:     "csrf_token",
	CookieMaxAge:   86400, // 24 hours
	CookieDomain:   "",
	CookiePath:     "/",
	CookieSecure:   true,
	CookieSameSite: http.SameSiteStrictMode,
	TokenLookup:    "header:X-CSRF-Token",
	ErrorHandler:   nil,
	ExemptPaths:    []string{},
	ExemptMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace},
	HMACKey:        nil,
}

// CSRFOption configures CSRF middleware
type CSRFOption func(*CSRFConfig)

// WithCSRFCookieName sets the CSRF cookie name
func WithCSRFCookieName(name string) CSRFOption {
	return func(c *CSRFConfig) {
		c.CookieName = name
	}
}

// WithCSRFCookieMaxAge sets the cookie max age in seconds
func WithCSRFCookieMaxAge(maxAge int) CSRFOption {
	return func(c *CSRFConfig) {
		c.CookieMaxAge = maxAge
	}
}

// WithCSRFCookieDomain sets the cookie domain
func WithCSRFCookieDomain(domain string) CSRFOption {
	return func(c *CSRFConfig) {
		c.CookieDomain = domain
	}
}

// WithCSRFCookiePath sets the cookie path
func WithCSRFCookiePath(path string) CSRFOption {
	return func(c *CSRFConfig) {
		c.CookiePath = path
	}
}

// WithCSRFCookieSecure sets the Secure flag
func WithCSRFCookieSecure(secure bool) CSRFOption {
	return func(c *CSRFConfig) {
		c.CookieSecure = secure
	}
}

// WithCSRFCookieSameSite sets the SameSite attribute
func WithCSRFCookieSameSite(sameSite http.SameSite) CSRFOption {
	return func(c *CSRFConfig) {
		c.CookieSameSite = sameSite
	}
}

// WithCSRFTokenLookup sets the token lookup format
func WithCSRFTokenLookup(lookup string) CSRFOption {
	return func(c *CSRFConfig) {
		c.TokenLookup = lookup
	}
}

// WithCSRFErrorHandler sets a custom error handler
func WithCSRFErrorHandler(handler http.HandlerFunc) CSRFOption {
	return func(c *CSRFConfig) {
		c.ErrorHandler = handler
	}
}

// WithCSRFExemptPaths sets exempt paths
func WithCSRFExemptPaths(paths []string) CSRFOption {
	return func(c *CSRFConfig) {
		c.ExemptPaths = paths
	}
}

// WithCSRFExemptMethods sets exempt HTTP methods
func WithCSRFExemptMethods(methods []string) CSRFOption {
	return func(c *CSRFConfig) {
		c.ExemptMethods = methods
	}
}

// WithCSRFHMACKey sets the HMAC signing key
func WithCSRFHMACKey(key []byte) CSRFOption {
	return func(c *CSRFConfig) {
		c.HMACKey = key
	}
}

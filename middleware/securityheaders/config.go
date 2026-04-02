package securityheaders

import "strings"

// https://www.permissionspolicy.com/
var permissionPolicyFeatures = []string{
	"accelerometer=()",
	"autoplay=()",
	"camera=()",
	"cross-origin-isolated=()",
	"display-capture=()",
	"encrypted-media=()",
	"fullscreen=()",
	"geolocation=()",
	"gyroscope=()",
	"keyboard-map=()",
	"magnetometer=()",
	"microphone=()",
	"midi=()",
	"payment=()",
	"picture-in-picture=()",
	"publickey-credentials-get=()",
	"screen-wake-lock=()",
	"sync-xhr=()",
	"usb=()",
	"web-share=()",
	"xr-spatial-tracking=()",
}

// StrictTransportSecurity defines the parameters for HTTP Strict Transport Security (HSTS).
type StrictTransportSecurity struct {
	// MaxAge sets the time, in seconds, that the browser should remember that a site is only to be accessed using HTTPS.
	// A value of 0 disables HSTS.
	// Default: 0 (disabled)
	MaxAge int

	// ExcludeSubdomains specifies whether the HSTS policy applies to all subdomains.
	// Default: false
	ExcludeSubdomains bool

	// PreloadEnabled adds the preload directive to the header.
	// Default: false
	PreloadEnabled bool
}

// DefaultStrictTransportSecurity contains default values for HSTS configuration.
var DefaultStrictTransportSecurity = StrictTransportSecurity{
	MaxAge:            0, // Disabled by default
	ExcludeSubdomains: false,
	PreloadEnabled:    false,
}

// cspNonceContextKey is the context key type for CSP nonce values.
type cspNonceContextKey struct{}

// CSPNonceContextKey is the context key for CSP nonces.
var CSPNonceContextKey = cspNonceContextKey{}

const (
	// CSPNoncePlaceholder is the placeholder string replaced with the actual nonce
	CSPNoncePlaceholder = "{{nonce}}"
)

// Config allows customization of security headers
type Config struct {
	// ContentSecurityPolicy sets the `Content-Security-Policy` header.
	// Use "{{nonce}}" placeholder to enable nonce generation (e.g., script-src 'nonce-{{nonce}}').
	// Default: "default-src 'none'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'self'; frame-ancestors 'self'; form-action 'self';"
	ContentSecurityPolicy string

	// ContentSecurityPolicyReportOnly sets the policy in report-only mode.
	// Default: false
	ContentSecurityPolicyReportOnly bool

	// ContentSecurityPolicyNonceEnabled generates a unique nonce per request when true.
	// The nonce replaces "{{nonce}}" in the CSP header and is stored in context.
	// Default: false
	ContentSecurityPolicyNonceEnabled bool

	// ContentSecurityPolicyNonceContextKey overrides the default context key for storing the nonce.
	// Default: CSPNonceContextKey
	ContentSecurityPolicyNonceContextKey any

	// CrossOriginEmbedderPolicy sets the `Cross-Origin-Embedder-Policy` header.
	// Default: "require-corp"
	CrossOriginEmbedderPolicy string

	// CrossOriginOpenerPolicy sets the `Cross-Origin-Opener-Policy` header.
	// Default: "same-origin"
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy sets the `Cross-Origin-Resource-Policy` header.
	// Default: "same-origin"
	CrossOriginResourcePolicy string

	// PermissionsPolicy sets the `Permissions-Policy` header.
	// Default: permission policy features (accelerometer, camera, geolocation, etc.)
	PermissionsPolicy string

	// ReferrerPolicy sets the `Referrer-Policy` header.
	// Default: "no-referrer"
	ReferrerPolicy string

	// Server sets the `Server` header. Use empty string to hide server info.
	// Default: "" (hidden)
	Server string

	// StrictTransportSecurity configures HSTS header.
	// Default: MaxAge=0 (disabled), ExcludeSubdomains=false, PreloadEnabled=false
	StrictTransportSecurity StrictTransportSecurity

	// XContentTypeOptions sets the `X-Content-Type-Options` header.
	// Default: "nosniff"
	XContentTypeOptions string

	// XFrameOptions sets the `X-Frame-Options` header.
	// Default: "DENY"
	XFrameOptions string

	// ExcludedPaths contains paths to skip security headers.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where security headers are explicitly applied.
	// If set, security headers will only be set for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, security headers apply to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default values for security headers configuration.
var DefaultConfig = Config{
	ContentSecurityPolicy:     "default-src 'none'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'self'; frame-ancestors 'self'; form-action 'self';",
	CrossOriginEmbedderPolicy: "require-corp",
	CrossOriginOpenerPolicy:   "same-origin",
	CrossOriginResourcePolicy: "same-origin",
	PermissionsPolicy:         strings.Join(permissionPolicyFeatures, ", "),
	ReferrerPolicy:            "no-referrer",
	StrictTransportSecurity:   DefaultStrictTransportSecurity,
	XContentTypeOptions:       "nosniff",
	XFrameOptions:             "DENY",
	ExcludedPaths:             []string{},
	IncludedPaths:             []string{},
}

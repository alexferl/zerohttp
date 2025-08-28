package config

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
	MaxAge int
	// ExcludeSubdomains specifies whether the HSTS policy applies to all subdomains.
	ExcludeSubdomains bool
	// PreloadEnabled adds the preload directive to the header.
	PreloadEnabled bool
}

// DefaultStrictTransportSecurity contains default values for HSTS configuration.
var DefaultStrictTransportSecurity = StrictTransportSecurity{
	MaxAge:            0, // Disabled by default
	ExcludeSubdomains: false,
	PreloadEnabled:    false,
}

// StrictTransportSecurityOption configures HSTS settings.
type StrictTransportSecurityOption func(*StrictTransportSecurity)

// WithHSTSMaxAge sets the max-age directive for HSTS.
func WithHSTSMaxAge(maxAge int) StrictTransportSecurityOption {
	return func(hsts *StrictTransportSecurity) {
		hsts.MaxAge = maxAge
	}
}

// WithHSTSExcludeSubdomains controls whether subdomains are excluded from HSTS.
func WithHSTSExcludeSubdomains(exclude bool) StrictTransportSecurityOption {
	return func(hsts *StrictTransportSecurity) {
		hsts.ExcludeSubdomains = exclude
	}
}

// WithHSTSPreload enables or disables the preload directive.
func WithHSTSPreload(preload bool) StrictTransportSecurityOption {
	return func(hsts *StrictTransportSecurity) {
		hsts.PreloadEnabled = preload
	}
}

// SecurityHeadersConfig allows customization of security headers
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy sets the `Content-Security-Policy` header
	ContentSecurityPolicy string
	// ContentSecurityPolicyReportOnly sets the policy in report-only mode
	ContentSecurityPolicyReportOnly bool
	// CrossOriginEmbedderPolicy sets the `Cross-Origin-Embedder-Policy` header
	CrossOriginEmbedderPolicy string
	// CrossOriginOpenerPolicy sets the `Cross-Origin-Opener-Policy` header
	CrossOriginOpenerPolicy string
	// CrossOriginResourcePolicy sets the `Cross-Origin-Resource-Policy` header
	CrossOriginResourcePolicy string
	// PermissionsPolicy sets the `Permissions-Policy` header
	PermissionsPolicy string
	// ReferrerPolicy sets the `Referrer-Policy` header
	ReferrerPolicy string
	// Server sets the `Server` header (empty string to hide server info)
	Server string
	// StrictTransportSecurity configures HSTS header
	StrictTransportSecurity StrictTransportSecurity
	// XContentTypeOptions sets the `X-Content-Type-Options` header
	XContentTypeOptions string
	// XFrameOptions sets the `X-Frame-Options` header
	XFrameOptions string
	// ExemptPaths contains paths to skip security headers
	ExemptPaths []string
}

// DefaultSecurityHeadersConfig contains the default values for security headers configuration.
var DefaultSecurityHeadersConfig = SecurityHeadersConfig{
	ContentSecurityPolicy:     "default-src 'none'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'self'; frame-ancestors 'self'; form-action 'self';",
	CrossOriginEmbedderPolicy: "require-corp",
	CrossOriginOpenerPolicy:   "same-origin",
	CrossOriginResourcePolicy: "same-origin",
	PermissionsPolicy:         strings.Join(permissionPolicyFeatures, ", "),
	ReferrerPolicy:            "no-referrer",
	StrictTransportSecurity:   DefaultStrictTransportSecurity,
	XContentTypeOptions:       "nosniff",
	XFrameOptions:             "DENY",
	ExemptPaths:               []string{},
}

// SecurityHeadersOption configures the security headers middleware.
type SecurityHeadersOption func(*SecurityHeadersConfig)

// WithSecurityHeadersCSP sets the Content-Security-Policy header.
func WithSecurityHeadersCSP(policy string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.ContentSecurityPolicy = policy
	}
}

// WithSecurityHeadersCSPReportOnly sets the policy in report-only mode.
func WithSecurityHeadersCSPReportOnly(reportOnly bool) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.ContentSecurityPolicyReportOnly = reportOnly
	}
}

// WithSecurityHeadersCrossOriginEmbedderPolicy sets the Cross-Origin-Embedder-Policy header.
func WithSecurityHeadersCrossOriginEmbedderPolicy(value string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.CrossOriginEmbedderPolicy = value
	}
}

// WithSecurityHeadersCrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header.
func WithSecurityHeadersCrossOriginOpenerPolicy(value string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.CrossOriginOpenerPolicy = value
	}
}

// WithSecurityHeadersCrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
func WithSecurityHeadersCrossOriginResourcePolicy(value string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.CrossOriginResourcePolicy = value
	}
}

// WithSecurityHeadersPermissionsPolicy sets the Permissions-Policy header.
func WithSecurityHeadersPermissionsPolicy(policy string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.PermissionsPolicy = policy
	}
}

// WithSecurityHeadersReferrerPolicy sets the Referrer-Policy header.
func WithSecurityHeadersReferrerPolicy(policy string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.ReferrerPolicy = policy
	}
}

// WithSecurityHeadersServer sets the Server header.
func WithSecurityHeadersServer(server string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.Server = server
	}
}

// WithSecurityHeadersHSTS configures the Strict-Transport-Security header.
func WithSecurityHeadersHSTS(opts ...StrictTransportSecurityOption) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		hsts := DefaultStrictTransportSecurity
		for _, opt := range opts {
			opt(&hsts)
		}
		c.StrictTransportSecurity = hsts
	}
}

// WithSecurityHeadersXContentTypeOptions sets the X-Content-Type-Options header.
func WithSecurityHeadersXContentTypeOptions(value string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.XContentTypeOptions = value
	}
}

// WithSecurityHeadersXFrameOptions sets the X-Frame-Options header.
func WithSecurityHeadersXFrameOptions(value string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.XFrameOptions = value
	}
}

// WithSecurityHeadersExemptPaths sets paths to skip security headers.
func WithSecurityHeadersExemptPaths(paths []string) SecurityHeadersOption {
	return func(c *SecurityHeadersConfig) {
		c.ExemptPaths = paths
	}
}

// securityHeadersConfigToOptions converts a SecurityHeadersConfig struct to a slice of SecurityHeadersOption functions.
func securityHeadersConfigToOptions(cfg SecurityHeadersConfig) []SecurityHeadersOption {
	return []SecurityHeadersOption{
		WithSecurityHeadersCSP(cfg.ContentSecurityPolicy),
		WithSecurityHeadersCSPReportOnly(cfg.ContentSecurityPolicyReportOnly),
		WithSecurityHeadersCrossOriginEmbedderPolicy(cfg.CrossOriginEmbedderPolicy),
		WithSecurityHeadersCrossOriginOpenerPolicy(cfg.CrossOriginOpenerPolicy),
		WithSecurityHeadersCrossOriginResourcePolicy(cfg.CrossOriginResourcePolicy),
		WithSecurityHeadersPermissionsPolicy(cfg.PermissionsPolicy),
		WithSecurityHeadersReferrerPolicy(cfg.ReferrerPolicy),
		WithSecurityHeadersServer(cfg.Server),
		WithSecurityHeadersHSTS(
			WithHSTSMaxAge(cfg.StrictTransportSecurity.MaxAge),
			WithHSTSExcludeSubdomains(cfg.StrictTransportSecurity.ExcludeSubdomains),
			WithHSTSPreload(cfg.StrictTransportSecurity.PreloadEnabled),
		),
		WithSecurityHeadersXContentTypeOptions(cfg.XContentTypeOptions),
		WithSecurityHeadersXFrameOptions(cfg.XFrameOptions),
		WithSecurityHeadersExemptPaths(cfg.ExemptPaths),
	}
}

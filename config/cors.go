package config

import "net/http"

// CORSConfig allows customization of CORS behavior
type CORSConfig struct {
	// AllowedOrigins is a list of allowed origins. Use ["*"] to allow all origins
	AllowedOrigins []string
	// AllowedMethods is a list of allowed HTTP methods (defaults to common methods)
	AllowedMethods []string
	// AllowedHeaders is a list of allowed request headers (defaults to common headers)
	AllowedHeaders []string
	// ExposedHeaders is a list of headers exposed to the client
	ExposedHeaders []string
	// AllowCredentials indicates whether credentials are allowed
	AllowCredentials bool
	// MaxAge indicates how long preflight requests can be cached (in seconds)
	MaxAge int
	// OptionsPassthrough allows OPTIONS requests to be passed to the next handler
	OptionsPassthrough bool
	// ExemptPaths contains paths that skip CORS processing
	ExemptPaths []string
}

// DefaultCORSConfig contains the default values for CORS configuration.
var DefaultCORSConfig = CORSConfig{
	AllowedOrigins: []string{"*"},
	AllowedMethods: []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	},
	AllowedHeaders: []string{
		"Accept",
		"Authorization",
		"Content-Type",
		"X-CSRF-Token",
		"X-Request-Id",
	},
	ExposedHeaders:     []string{},
	AllowCredentials:   false,
	MaxAge:             86400, // 24 hours
	OptionsPassthrough: false,
	ExemptPaths:        []string{},
}

// CORSOption configures CORS middleware.
type CORSOption func(*CORSConfig)

// WithCORSAllowedOrigins sets the list of allowed origins.
func WithCORSAllowedOrigins(origins []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedOrigins = origins
	}
}

// WithCORSAllowedMethods sets the list of allowed HTTP methods.
func WithCORSAllowedMethods(methods []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedMethods = methods
	}
}

// WithCORSAllowedHeaders sets the list of allowed request headers.
func WithCORSAllowedHeaders(headers []string) CORSOption {
	return func(c *CORSConfig) {
		c.AllowedHeaders = headers
	}
}

// WithCORSExposedHeaders sets the list of headers exposed to the client.
func WithCORSExposedHeaders(headers []string) CORSOption {
	return func(c *CORSConfig) {
		c.ExposedHeaders = headers
	}
}

// WithCORSAllowCredentials sets whether credentials are allowed.
func WithCORSAllowCredentials(allow bool) CORSOption {
	return func(c *CORSConfig) {
		c.AllowCredentials = allow
	}
}

// WithCORSMaxAge sets how long preflight requests can be cached (in seconds).
func WithCORSMaxAge(maxAge int) CORSOption {
	return func(c *CORSConfig) {
		c.MaxAge = maxAge
	}
}

// WithCORSOptionsPassthrough sets whether OPTIONS requests are passed to the next handler.
func WithCORSOptionsPassthrough(passthrough bool) CORSOption {
	return func(c *CORSConfig) {
		c.OptionsPassthrough = passthrough
	}
}

// WithCORSExemptPaths sets paths that skip CORS processing.
func WithCORSExemptPaths(paths []string) CORSOption {
	return func(c *CORSConfig) {
		c.ExemptPaths = paths
	}
}

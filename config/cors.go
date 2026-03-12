package config

import "net/http"

// OriginValidator is a function that validates if an origin is allowed.
// Returns true if the origin is allowed, false otherwise.
type OriginValidator func(origin string) bool

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

	// AllowOriginFunc is a custom function to validate origins dynamically.
	// If set, this takes precedence over AllowedOrigins matching.
	AllowOriginFunc OriginValidator
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

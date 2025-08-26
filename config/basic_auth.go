package config

// BasicAuthConfig allows customization of basic authentication
type BasicAuthConfig struct {
	// Realm is the authentication realm (defaults to "Restricted")
	Realm string
	// Credentials is a map of username -> password
	Credentials map[string]string
	// Validator is a custom function to validate credentials (optional)
	Validator func(username, password string) bool
	// ExemptPaths contains paths that skip basic auth (e.g., /health, /login, /signup)
	ExemptPaths []string
}

// DefaultBasicAuthConfig contains the default basic authentication configuration
var DefaultBasicAuthConfig = BasicAuthConfig{
	Realm:       "Restricted",
	Credentials: nil,
	Validator:   nil,
	ExemptPaths: []string{},
}

// BasicAuthOption configures basic authentication middleware
type BasicAuthOption func(*BasicAuthConfig)

// WithBasicAuthRealm sets the authentication realm
func WithBasicAuthRealm(realm string) BasicAuthOption {
	return func(c *BasicAuthConfig) {
		c.Realm = realm
	}
}

// WithBasicAuthCredentials sets username/password pairs
func WithBasicAuthCredentials(credentials map[string]string) BasicAuthOption {
	return func(c *BasicAuthConfig) {
		c.Credentials = credentials
	}
}

// WithBasicAuthValidator sets a custom credential validator function
func WithBasicAuthValidator(validator func(string, string) bool) BasicAuthOption {
	return func(c *BasicAuthConfig) {
		c.Validator = validator
	}
}

// WithBasicAuthExemptPaths sets paths that skip authentication
func WithBasicAuthExemptPaths(paths []string) BasicAuthOption {
	return func(c *BasicAuthConfig) {
		c.ExemptPaths = paths
	}
}

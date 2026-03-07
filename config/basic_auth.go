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

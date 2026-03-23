package basicauth

// Config allows customization of basic authentication
type Config struct {
	// Realm is the authentication realm (defaults to "Restricted")
	Realm string

	// Credentials is a map of username -> password
	Credentials map[string]string

	// Validator is a custom function to validate credentials (optional)
	Validator func(username, password string) bool

	// ExcludedPaths contains paths that skip basic auth.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where basic auth is explicitly applied.
	// If set, basic auth will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, basic auth applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default basic authentication configuration
var DefaultConfig = Config{
	Realm:         "Restricted",
	Credentials:   nil,
	Validator:     nil,
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
}

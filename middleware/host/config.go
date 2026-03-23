package host

// Config allows customization of Host header validation
type Config struct {
	// AllowedHosts is a list of allowed host values (e.g., "api.example.com", "example.com")
	// If empty, all hosts are allowed (no validation)
	AllowedHosts []string

	// AllowSubdomains allows any subdomain of the hosts in AllowedHosts
	// For example, if "example.com" is in AllowedHosts and AllowSubdomains is true,
	// then "api.example.com", "www.example.com", etc. are all valid
	AllowSubdomains bool

	// StrictPort requires the Host header to include a port if the server is
	// running on a non-standard port (i.e., not 80 or 443)
	// Requires Port to be set to the server's port
	StrictPort bool

	// Port is the port the server is listening on (e.g., 8080)
	// Used with StrictPort to validate the Host header includes the port
	Port int

	// StatusCode is the HTTP status code returned for invalid hosts
	// Defaults to 400 (Bad Request)
	StatusCode int

	// Message is the error message returned for invalid hosts
	// Defaults to "Invalid Host header"
	Message string

	// ExcludedPaths contains paths to skip host validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where host validation is explicitly applied.
	// If set, host validation will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, host validation applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains default values for host validation
var DefaultConfig = Config{
	AllowedHosts:    []string{},
	AllowSubdomains: false,
	StrictPort:      false,
	StatusCode:      400,
	Message:         "Invalid Host header",
	ExcludedPaths:   []string{},
	IncludedPaths:   []string{},
}

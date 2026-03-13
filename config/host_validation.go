package config

// HostValidationConfig allows customization of Host header validation
type HostValidationConfig struct {
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

	// ExemptPaths contains paths to skip host validation
	ExemptPaths []string
}

// DefaultHostValidationConfig contains default values for host validation
var DefaultHostValidationConfig = HostValidationConfig{
	AllowedHosts:    []string{},
	AllowSubdomains: false,
	StrictPort:      false,
	StatusCode:      400,
	Message:         "Invalid Host header",
	ExemptPaths:     []string{},
}

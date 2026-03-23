package contenttype

import "github.com/alexferl/zerohttp/httpx"

// Config allows customization of allowed content types
type Config struct {
	// ContentTypes is a list of allowed content types
	ContentTypes []string

	// ExcludedPaths contains paths that skip content type validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where content type validation is explicitly applied.
	// If set, validation will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, validation applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default values for content type configuration.
var DefaultConfig = Config{
	ContentTypes: []string{
		httpx.MIMEApplicationJSON,
		httpx.MIMEApplicationFormURLEncoded,
		httpx.MIMEMultipartFormData,
	},
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
}

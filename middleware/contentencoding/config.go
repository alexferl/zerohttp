package contentencoding

import "github.com/alexferl/zerohttp/httpx"

// Config allows customization of allowed content encodings
type Config struct {
	// Encodings is a list of allowed content encodings (gzip, deflate, br, etc.).
	// Default: [gzip, deflate]
	Encodings []string

	// ExcludedPaths contains paths that skip content encoding validation.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where content encoding validation is explicitly applied.
	// If set, validation will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, validation applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default values for content encoding configuration.
var DefaultConfig = Config{
	Encodings:     []string{httpx.ContentEncodingGzip, httpx.ContentEncodingDeflate},
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
}

package requestbodysize

// Config allows customization of request size limiting.
type Config struct {
	// MaxBytes is the maximum request body size in bytes.
	// Default: 1MB
	MaxBytes int64

	// ExcludedPaths contains paths that skip body size limiting.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where body size limiting is explicitly applied.
	// If set, body size limiting will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, body size limiting applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string
}

// DefaultConfig contains the default values for request body size limiting.
var DefaultConfig = Config{
	MaxBytes:      1 << 20, // 1MB default
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
}

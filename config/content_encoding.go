package config

// ContentEncodingConfig allows customization of allowed content encodings
type ContentEncodingConfig struct {
	// Encodings is a list of allowed content encodings (gzip, deflate, br, etc.)
	Encodings []string
	// ExemptPaths contains paths that skip content encoding validation
	ExemptPaths []string
}

// DefaultContentEncodingConfig contains the default values for content encoding configuration.
var DefaultContentEncodingConfig = ContentEncodingConfig{
	Encodings:   []string{"gzip", "deflate"},
	ExemptPaths: []string{},
}

// ContentEncodingOption configures content encoding middleware.
type ContentEncodingOption func(*ContentEncodingConfig)

// WithContentEncodingEncodings sets the list of allowed content encodings.
func WithContentEncodingEncodings(encodings []string) ContentEncodingOption {
	return func(c *ContentEncodingConfig) {
		c.Encodings = encodings
	}
}

// WithContentEncodingExemptPaths sets paths that skip content encoding validation.
func WithContentEncodingExemptPaths(paths []string) ContentEncodingOption {
	return func(c *ContentEncodingConfig) {
		c.ExemptPaths = paths
	}
}

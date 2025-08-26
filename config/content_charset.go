package config

// ContentCharsetConfig allows customization of allowed charsets
type ContentCharsetConfig struct {
	// Charsets is a list of allowed character encodings
	// An empty string allows requests with no charset specified
	Charsets []string
}

// DefaultContentCharsetConfig contains the default values for content charset configuration.
var DefaultContentCharsetConfig = ContentCharsetConfig{
	Charsets: []string{"utf-8", ""}, // Default allows UTF-8 and no charset
}

// ContentCharsetOption configures content charset middleware.
type ContentCharsetOption func(*ContentCharsetConfig)

// WithContentCharsetCharsets sets the list of allowed character encodings.
func WithContentCharsetCharsets(charsets []string) ContentCharsetOption {
	return func(c *ContentCharsetConfig) {
		c.Charsets = charsets
	}
}

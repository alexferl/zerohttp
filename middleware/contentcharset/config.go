package contentcharset

// Config allows customization of allowed charsets
type Config struct {
	// Charsets is a list of allowed character encodings
	// An empty string allows requests with no charset specified
	Charsets []string
}

// DefaultConfig contains the default values for content charset configuration.
var DefaultConfig = Config{
	Charsets: []string{"utf-8", ""}, // Default allows UTF-8 and no charset
}

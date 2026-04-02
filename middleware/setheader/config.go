package setheader

// Config allows customization of response headers
type Config struct {
	// Headers is a map of header key-value pairs to set.
	// Default: {} (empty map)
	Headers map[string]string
}

// DefaultConfig contains the default values for set header configuration.
var DefaultConfig = Config{
	Headers: make(map[string]string),
}

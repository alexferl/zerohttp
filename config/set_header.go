package config

// SetHeaderConfig allows customization of response headers
type SetHeaderConfig struct {
	// Headers is a map of header key-value pairs to set
	Headers map[string]string
}

// DefaultSetHeaderConfig contains the default values for set header configuration.
var DefaultSetHeaderConfig = SetHeaderConfig{
	Headers: make(map[string]string),
}

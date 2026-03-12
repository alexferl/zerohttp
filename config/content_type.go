package config

// ContentTypeConfig allows customization of allowed content types
type ContentTypeConfig struct {
	// ContentTypes is a list of allowed content types
	ContentTypes []string

	// ExemptPaths contains paths that skip content type validation
	ExemptPaths []string
}

// DefaultContentTypeConfig contains the default values for content type configuration.
var DefaultContentTypeConfig = ContentTypeConfig{
	ContentTypes: []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	ExemptPaths:  []string{},
}

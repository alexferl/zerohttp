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

// ContentTypeOption configures content type middleware.
type ContentTypeOption func(*ContentTypeConfig)

// WithContentTypeContentTypes sets the list of allowed content types.
func WithContentTypeContentTypes(contentTypes []string) ContentTypeOption {
	return func(c *ContentTypeConfig) {
		c.ContentTypes = contentTypes
	}
}

// WithContentTypeExemptPaths sets paths that skip content type validation.
func WithContentTypeExemptPaths(paths []string) ContentTypeOption {
	return func(c *ContentTypeConfig) {
		c.ExemptPaths = paths
	}
}

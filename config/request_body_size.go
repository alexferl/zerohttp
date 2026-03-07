package config

// RequestBodySizeConfig allows customization of request size limiting.
type RequestBodySizeConfig struct {
	// MaxBytes is the maximum request body size in bytes.
	MaxBytes int64
	// ExemptPaths contains paths that skip body size limiting.
	ExemptPaths []string
}

// DefaultRequestBodySizeConfig contains the default values for request body size limiting.
var DefaultRequestBodySizeConfig = RequestBodySizeConfig{
	MaxBytes:    1 << 20, // 1MB default
	ExemptPaths: []string{},
}

// RequestBodySizeOption configures request body size limiting middleware.
type RequestBodySizeOption func(*RequestBodySizeConfig)

// WithRequestBodySizeMaxBytes sets the maximum allowed body size in bytes.
func WithRequestBodySizeMaxBytes(size int64) RequestBodySizeOption {
	return func(c *RequestBodySizeConfig) {
		c.MaxBytes = size
	}
}

// WithRequestBodySizeExemptPaths sets paths that skip body size limiting.
func WithRequestBodySizeExemptPaths(paths []string) RequestBodySizeOption {
	return func(c *RequestBodySizeConfig) {
		c.ExemptPaths = paths
	}
}

// requestBodySizeConfigToOptions converts a RequestBodySizeConfig struct to a slice of RequestBodySizeOption functions.
func requestBodySizeConfigToOptions(cfg RequestBodySizeConfig) []RequestBodySizeOption {
	return []RequestBodySizeOption{
		WithRequestBodySizeMaxBytes(cfg.MaxBytes),
		WithRequestBodySizeExemptPaths(cfg.ExemptPaths),
	}
}

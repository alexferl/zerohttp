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

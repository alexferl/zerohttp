package config

// RecoverConfig allows customization of panic recovery
type RecoverConfig struct {
	// StackSize is the maximum size of the stack trace in bytes (defaults to 4KB)
	StackSize int64
	// EnableStackTrace determines if stack traces should be included (defaults to true)
	EnableStackTrace bool
}

// DefaultRecoverConfig contains the default panic recovery configuration
var DefaultRecoverConfig = RecoverConfig{
	StackSize:        4 << 10, // 4KB
	EnableStackTrace: true,
}

// recoverConfigToOptions converts a RecoverConfig struct to a slice of RecoverOption functions
func recoverConfigToOptions(cfg RecoverConfig) []RecoverOption {
	return []RecoverOption{
		WithRecoverStackSize(cfg.StackSize),
		WithRecoverEnableStackTrace(cfg.EnableStackTrace),
	}
}

// RecoverOption configures panic recovery middleware
type RecoverOption func(*RecoverConfig)

// WithRecoverStackSize sets the maximum stack trace size in bytes
func WithRecoverStackSize(size int64) RecoverOption {
	return func(c *RecoverConfig) {
		c.StackSize = size
	}
}

// WithRecoverEnableStackTrace enables or disables stack trace inclusion
func WithRecoverEnableStackTrace(enabled bool) RecoverOption {
	return func(c *RecoverConfig) {
		c.EnableStackTrace = enabled
	}
}

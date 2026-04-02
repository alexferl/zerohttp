package recover

// Config allows customization of panic recovery
type Config struct {
	// StackSize is the maximum size of the stack trace in bytes.
	// Default: 4KB
	StackSize int64

	// EnableStackTrace determines if stack traces should be included.
	// Default: true
	EnableStackTrace bool

	// RequestIDHeader is the header name for the request ID.
	// This should match the header configured in RequestIDConfig.
	// Default: "X-Request-Id"
	RequestIDHeader string
}

// DefaultConfig contains the default panic recovery configuration
var DefaultConfig = Config{
	StackSize:        4 << 10, // 4KB
	EnableStackTrace: true,
	RequestIDHeader:  "X-Request-Id",
}

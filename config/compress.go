package config

// CompressionAlgorithm defines supported compression algorithms
type CompressionAlgorithm string

const (
	Gzip    CompressionAlgorithm = "gzip"
	Deflate CompressionAlgorithm = "deflate"
)

// CompressConfig allows customization of compression behavior
type CompressConfig struct {
	// Level is the compression level (1-9 for gzip/deflate)
	Level int
	// Types are MIME types to compress (defaults to common text types)
	Types []string
	// Algorithms are compression algorithms to support (defaults to gzip, deflate)
	Algorithms []CompressionAlgorithm
	// ExemptPaths contains paths to skip compression
	ExemptPaths []string
}

// DefaultCompressConfig contains the default values for compression configuration.
var DefaultCompressConfig = CompressConfig{
	Level: 6,
	Types: []string{
		"text/html",
		"text/css",
		"text/plain",
		"text/javascript",
		"application/javascript",
		"application/json",
		"application/xml",
		"text/xml",
		"application/rss+xml",
		"application/atom+xml",
		"image/svg+xml",
	},
	Algorithms:  []CompressionAlgorithm{Gzip, Deflate},
	ExemptPaths: []string{},
}

// CompressOption configures compression middleware.
type CompressOption func(*CompressConfig)

// WithCompressLevel sets the compression level (1-9 for gzip/deflate).
func WithCompressLevel(level int) CompressOption {
	return func(c *CompressConfig) {
		c.Level = level
	}
}

// WithCompressTypes sets the MIME types to compress.
func WithCompressTypes(types []string) CompressOption {
	return func(c *CompressConfig) {
		c.Types = types
	}
}

// WithCompressAlgorithms sets the compression algorithms to support.
func WithCompressAlgorithms(algorithms []CompressionAlgorithm) CompressOption {
	return func(c *CompressConfig) {
		c.Algorithms = algorithms
	}
}

// WithCompressExemptPaths sets paths to skip compression.
func WithCompressExemptPaths(paths []string) CompressOption {
	return func(c *CompressConfig) {
		c.ExemptPaths = paths
	}
}

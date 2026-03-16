package config

import (
	"io"

	"github.com/alexferl/zerohttp/httpx"
)

// CompressionAlgorithm defines supported compression algorithms
type CompressionAlgorithm string

const (
	Gzip    = CompressionAlgorithm(httpx.ContentEncodingGzip)
	Deflate = CompressionAlgorithm(httpx.ContentEncodingDeflate)
)

// CompressionEncoder is the interface for compression encoders.
// Users can implement this interface to provide custom compression algorithms
// (e.g., Brotli, zstd) without adding dependencies to the core library.
//
// Example with github.com/andybalholm/brotli:
//
//	type BrotliEncoder struct{}
//	func (e BrotliEncoder) Encode(w io.Writer, level int) io.Writer {
//		return brotli.NewWriterLevel(w, level)
//	}
//	func (e BrotliEncoder) Encoding() string { return consts.ContentEncodingBrotli }
type CompressionEncoder interface {
	// Encode wraps the provided io.Writer with compression.
	// The level parameter is algorithm-specific (e.g., 1-9 for gzip, 0-11 for brotli).
	Encode(w io.Writer, level int) io.Writer

	// Encoding returns the encoding name used in Accept-Encoding/Content-Encoding headers.
	// Common values: consts.ContentEncodingGzip, consts.ContentEncodingDeflate, consts.ContentEncodingBrotli (brotli), consts.ContentEncodingZstd.
	Encoding() string
}

// CompressionProvider creates compression encoders.
// Implement this interface to bring your own compression algorithms.
//
// Example:
//
//	type MyCompressorProvider struct{}
//	func (p MyCompressorProvider) GetEncoder(encoding string) CompressionEncoder {
//	    switch encoding {
//	    case consts.ContentEncodingBrotli:
//	        return BrotliEncoder{}
//	    case consts.ContentEncodingZstd:
//	        return ZstdEncoder{}
//	    }
//	    return nil
//	}
//
//	cfg := config.CompressConfig{
//	    Provider: MyCompressorProvider{},
//	}
type CompressionProvider interface {
	// GetEncoder returns a CompressionEncoder for the given encoding, or nil if not supported.
	// The encoding will be lowercase (e.g., consts.ContentEncodingGzip, consts.ContentEncodingBrotli, consts.ContentEncodingZstd).
	GetEncoder(encoding string) CompressionEncoder
}

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

	// Provider is an optional custom compression provider.
	// If set, the provider's encoders will be used in addition to built-in gzip/deflate.
	// This allows users to add Brotli, zstd, or other algorithms without core dependencies.
	// Default: nil (use only built-in gzip/deflate)
	Provider CompressionProvider
}

// DefaultCompressConfig contains the default values for compression configuration.
var DefaultCompressConfig = CompressConfig{
	Level: 6,
	Types: []string{
		httpx.MIMETextHTML,
		"text/css",
		httpx.MIMETextPlain,
		"text/javascript",
		"application/javascript",
		httpx.MIMEApplicationJSON,
		"application/xml",
		"text/xml",
		"application/rss+xml",
		"application/atom+xml",
		"image/svg+xml",
	},
	Algorithms:  []CompressionAlgorithm{Gzip, Deflate},
	ExemptPaths: []string{},
}

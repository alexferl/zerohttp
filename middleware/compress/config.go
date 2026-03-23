package compress

import (
	"io"

	"github.com/alexferl/zerohttp/httpx"
)

// Algorithm defines supported compression algorithms
type Algorithm string

const (
	Gzip    = Algorithm(httpx.ContentEncodingGzip)
	Deflate = Algorithm(httpx.ContentEncodingDeflate)
)

// DefaultCompressTypes are the MIME types compressed by default.
var DefaultCompressTypes = []string{
	httpx.MIMETextHTML,
	httpx.MIMETextCSS,
	httpx.MIMETextPlain,
	httpx.MIMETextJavaScript,
	httpx.MIMEApplicationJavaScript,
	httpx.MIMEApplicationJSON,
	httpx.MIMEApplicationXML,
	httpx.MIMETextXML,
	httpx.MIMEApplicationRSSXML,
	httpx.MIMEApplicationAtomXML,
	httpx.MIMEImageSVGXML,
}

// Encoder is the interface for compression encoders.
// Users can implement this interface to provide custom compression algorithms
// (e.g., Brotli, zstd) without adding dependencies to the core library.
//
// Example with github.com/andybalholm/brotli:
//
//	type BrotliEncoder struct{}
//	func (e BrotliEncoder) Encode(w io.Writer, level int) io.Writer {
//		return brotli.NewWriterLevel(w, level)
//	}
//	func (e BrotliEncoder) Encoding() string { return httpx.ContentEncodingBrotli }
type Encoder interface {
	// Encode wraps the provided io.Writer with compression.
	// The level parameter is algorithm-specific (e.g., 1-9 for gzip, 0-11 for brotli).
	Encode(w io.Writer, level int) io.Writer

	// Encoding returns the encoding name used in Accept-Encoding/Content-Encoding headers.
	// Common values: httpx.ContentEncodingGzip, httpx.ContentEncodingDeflate, httpx.ContentEncodingBrotli (brotli), httpx.ContentEncodingZstd.
	Encoding() string
}

// Provider creates compression encoders.
// Implement this interface to bring your own compression algorithms.
//
// Example:
//
//	type MyCompressorProvider struct{}
//	func (p MyCompressorProvider) GetEncoder(encoding string) CompressionEncoder {
//	    switch encoding {
//	    case httpx.ContentEncodingBrotli:
//	        return BrotliEncoder{}
//	    case httpx.ContentEncodingZstd:
//	        return ZstdEncoder{}
//	    }
//	    return nil
//	}
//
//	cfg := config.CompressConfig{
//	    Providers: []config.CompressionProvider{MyCompressorProvider{}},
//	}
type Provider interface {
	// GetEncoder returns a Encoder for the given encoding, or nil if not supported.
	// The encoding will be lowercase (e.g., httpx.ContentEncodingGzip, httpx.ContentEncodingBrotli, httpx.ContentEncodingZstd).
	GetEncoder(encoding string) Encoder
}

// Config allows customization of compression behavior
type Config struct {
	// Level is the compression level (1-9 for gzip/deflate)
	Level int

	// Types are MIME types to compress (defaults to common text types)
	Types []string

	// Algorithms are compression algorithms to support (defaults to gzip, deflate).
	// The order determines precedence when the client accepts multiple encodings -
	// earlier algorithms are preferred over later ones.
	Algorithms []Algorithm

	// ExcludedPaths contains paths to skip compression.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where compression is explicitly applied.
	// If set, compression will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, compression applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// Providers are optional custom compression providers.
	// If set, the providers' encoders will be used in addition to built-in gzip/deflate.
	// This allows users to add Brotli, zstd, or other algorithms without core dependencies.
	// Default: nil (use only built-in gzip/deflate)
	Providers []Provider
}

// DefaultConfig contains the default values for compression configuration.
var DefaultConfig = Config{
	Level:         6,
	Types:         DefaultCompressTypes,
	Algorithms:    []Algorithm{Gzip, Deflate},
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
}

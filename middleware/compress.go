package middleware

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/metrics"
)

// Compressor represents a set of encoding configurations.
type Compressor struct {
	// The mapping of encoder names to encoder functions.
	encoders map[string]EncoderFunc
	// The mapping of pooled encoders to pools.
	pooledEncoders map[string]*sync.Pool
	// The set of content types allowed to be compressed.
	allowedTypes     map[string]struct{}
	allowedWildcards map[string]struct{}
	// The list of encoders in order of decreasing precedence.
	encodingPrecedence []string
	level              int                                  // The compression level.
	algorithms         map[config.CompressionAlgorithm]bool // Allowed algorithms
	algorithmOrder     []config.CompressionAlgorithm        // Algorithm precedence order
	excludedPaths      []string                             // Paths to skip compression
	includedPaths      []string                             // Paths to allow compression (if set, only these paths)
}

// NewCompressor creates a new Compressor that will handle encoding responses.
func NewCompressor(level int, types ...string) *Compressor {
	allowedTypes := make(map[string]struct{})
	allowedWildcards := make(map[string]struct{})
	if len(types) > 0 {
		for _, t := range types {
			if strings.Contains(strings.TrimSuffix(t, "/*"), "*") {
				panic(fmt.Sprintf("middleware/compress: Unsupported content-type wildcard pattern '%s'. Only '/*' supported", t))
			}
			if strings.HasSuffix(t, "/*") {
				allowedWildcards[strings.TrimSuffix(t, "/*")] = struct{}{}
			} else {
				allowedTypes[t] = struct{}{}
			}
		}
	} else {
		for _, t := range config.DefaultCompressConfig.Types {
			allowedTypes[t] = struct{}{}
		}
	}
	c := &Compressor{
		level:            level,
		encoders:         make(map[string]EncoderFunc),
		pooledEncoders:   make(map[string]*sync.Pool),
		allowedTypes:     allowedTypes,
		allowedWildcards: allowedWildcards,
		algorithms:       make(map[config.CompressionAlgorithm]bool),
		excludedPaths:    []string{},
		includedPaths:    []string{},
	}

	// Set default algorithms
	c.algorithms[config.Gzip] = true
	c.algorithms[config.Deflate] = true
	c.algorithmOrder = []config.CompressionAlgorithm{config.Gzip, config.Deflate}

	// Register encoders in algorithm order (first = highest precedence)
	for _, alg := range c.algorithmOrder {
		switch alg {
		case config.Gzip:
			c.SetEncoder(httpx.ContentEncodingGzip, encoderGzip)
		case config.Deflate:
			c.SetEncoder(httpx.ContentEncodingDeflate, encoderDeflate)
		}
	}
	return c
}

// SetEncoder can be used to set the implementation of a compression algorithm.
func (c *Compressor) SetEncoder(encoding string, fn EncoderFunc) {
	encoding = strings.ToLower(encoding)
	if encoding == "" {
		panic("the encoding can not be empty")
	}
	if fn == nil {
		panic("attempted to set a nil encoder function")
	}

	delete(c.pooledEncoders, encoding)
	delete(c.encoders, encoding)

	encoder := fn(io.Discard, c.level)
	if _, ok := encoder.(ioResetterWriter); ok {
		pool := &sync.Pool{
			New: func() any {
				return fn(io.Discard, c.level)
			},
		}
		c.pooledEncoders[encoding] = pool
	}

	if _, ok := c.pooledEncoders[encoding]; !ok {
		c.encoders[encoding] = fn
	}

	for i, v := range c.encodingPrecedence {
		if v == encoding {
			c.encodingPrecedence = append(c.encodingPrecedence[:i], c.encodingPrecedence[i+1:]...)
		}
	}
	c.encodingPrecedence = append([]string{encoding}, c.encodingPrecedence...)
}

// isExcludedPath checks if a path should be excluded from compression
func (c *Compressor) isExcludedPath(path string) bool {
	return !shouldProcessMiddleware(path, c.includedPaths, c.excludedPaths)
}

// Handler returns a new middleware that will compress the response.
func (c *Compressor) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check excluded paths first
		if c.isExcludedPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

		encoder, encoding, cleanup := c.selectEncoder(r.Header, w)
		isHead := r.Method == http.MethodHead
		cw := &compressResponseWriter{
			ResponseWriter:   w,
			w:                w,
			contentTypes:     c.allowedTypes,
			contentWildcards: c.allowedWildcards,
			encoding:         encoding,
			compressible:     false,
			isHeadRequest:    isHead,
		}
		// Don't use encoder for HEAD requests - it would set incorrect Content-Length
		if encoder != nil && !isHead {
			cw.w = encoder
		}

		// Only cleanup/close encoder if we used it
		if !isHead {
			defer cleanup()
			defer func() {
				_ = cw.Close()
			}()
		}
		defer func() {
			// Flush headers if not already flushed (e.g., for HEAD requests that don't write body)
			cw.flushHeader()
			// Record metric for encoding used
			enc := encoding
			if enc == "" {
				enc = "none"
			}
			reg.Counter("compress_requests_total", "encoding").WithLabelValues(enc).Inc()
		}()

		next.ServeHTTP(cw, r)
	})
}

// selectEncoder returns the encoder, the name of the encoder, and a closer function.
// Uses algorithmOrder to determine precedence (first = highest priority).
func (c *Compressor) selectEncoder(h http.Header, w io.Writer) (io.Writer, string, func()) {
	header := h.Get(httpx.HeaderAcceptEncoding)
	accepted := strings.Split(strings.ToLower(header), ",")

	// Iterate through algorithms in configured order (highest precedence first)
	for _, alg := range c.algorithmOrder {
		name := strings.ToLower(string(alg))
		if !matchAcceptEncoding(accepted, name) {
			continue
		}

		// Check if algorithm is allowed
		if !c.algorithms[alg] {
			continue
		}

		if pool, ok := c.pooledEncoders[name]; ok {
			encoder := pool.Get().(ioResetterWriter)
			cleanup := func() {
				pool.Put(encoder)
			}
			encoder.Reset(w)
			return encoder, name, cleanup
		}
		if fn, ok := c.encoders[name]; ok {
			encoder := fn(w, c.level)
			if encoder == nil {
				continue // Skip if encoder failed to create (invalid level)
			}
			return encoder, name, func() {}
		}
	}
	return nil, "", func() {}
}

func matchAcceptEncoding(accepted []string, encoding string) bool {
	for _, v := range accepted {
		name, params, _ := strings.Cut(strings.TrimSpace(v), ";")
		name = strings.TrimSpace(name)
		if name != encoding && name != "*" {
			continue
		}
		// q=0 means explicitly not acceptable (RFC 7231 §5.3.1)
		if q, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if q == "0" || q == "0.0" || q == "0.00" || q == "0.000" {
				return false
			}
		}
		return true
	}
	return false
}

// EncoderFunc is a function that wraps the provided io.Writer with compression.
type EncoderFunc func(w io.Writer, level int) io.Writer

// Interface for types that allow resetting io.Writers.
type ioResetterWriter interface {
	io.Writer
	Reset(w io.Writer)
}

type compressResponseWriter struct {
	http.ResponseWriter
	w                io.Writer
	contentTypes     map[string]struct{}
	contentWildcards map[string]struct{}
	encoding         string
	wroteHeader      bool
	compressible     bool
	isHeadRequest    bool
	statusCode       int  // stored status for deferred WriteHeader
	headerFlushed    bool // whether headers have been flushed to underlying ResponseWriter
}

func (cw *compressResponseWriter) isCompressible() bool {
	contentType := cw.Header().Get(httpx.HeaderContentType)
	contentType, _, _ = strings.Cut(contentType, ";")

	if _, ok := cw.contentTypes[contentType]; ok {
		return true
	}
	if contentType, _, hadSlash := strings.Cut(contentType, "/"); hadSlash {
		_, ok := cw.contentWildcards[contentType]
		return ok
	}
	return false
}

func (cw *compressResponseWriter) WriteHeader(code int) {
	if cw.wroteHeader {
		// Ignore subsequent WriteHeader calls, matching standard library behavior
		return
	}
	cw.wroteHeader = true
	cw.statusCode = code
	// Don't call underlying WriteHeader yet - defer until Write() is called
	// This allows handlers to set status before writing body without breaking compression
}

// flushHeader writes the stored status and headers to the underlying ResponseWriter.
// This is called on the first Write() to ensure headers are sent with the correct status.
func (cw *compressResponseWriter) flushHeader() {
	if cw.headerFlushed {
		return
	}
	cw.headerFlushed = true

	code := cw.statusCode
	if code == 0 {
		code = http.StatusOK
	}

	if cw.Header().Get(httpx.HeaderContentEncoding) == "" && cw.encoding != "" {
		isCompressible := cw.isCompressible()
		contentType := cw.Header().Get(httpx.HeaderContentType)

		// Set Content-Encoding header if:
		// 1. Content is compressible, OR
		// 2. No Content-Type is set (e.g., HEAD request where type isn't determined yet)
		if isCompressible || contentType == "" {
			cw.Header().Set(httpx.HeaderContentEncoding, cw.encoding)
			cw.Header().Add(httpx.HeaderVary, httpx.HeaderAcceptEncoding)
			cw.Header().Del(httpx.HeaderContentLength)
		}

		if isCompressible {
			cw.compressible = true
		}
	}

	cw.ResponseWriter.WriteHeader(code)
}

func (cw *compressResponseWriter) Write(p []byte) (int, error) {
	cw.flushHeader()
	// For HEAD requests, don't write body to response.
	// We can't set Content-Length correctly because:
	// 1. WriteHeader is already called by this point
	// 2. We don't know compressed size without actually compressing
	if cw.isHeadRequest {
		return len(p), nil
	}
	return cw.writer().Write(p)
}

func (cw *compressResponseWriter) writer() io.Writer {
	if cw.compressible {
		return cw.w
	}
	return cw.ResponseWriter
}

type compressFlusher interface {
	Flush() error
}

func (cw *compressResponseWriter) Flush() {
	if f, ok := cw.writer().(http.Flusher); ok {
		f.Flush()
	}
	if f, ok := cw.writer().(compressFlusher); ok {
		_ = f.Flush()
		if f, ok := cw.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func (cw *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := cw.writer().(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("middleware: http.Hijacker is unavailable on the writer")
}

func (cw *compressResponseWriter) Push(target string, opts *http.PushOptions) error {
	if ps, ok := cw.writer().(http.Pusher); ok {
		return ps.Push(target, opts)
	}
	return errors.New("middleware: http.Pusher is unavailable on the writer")
}

func (cw *compressResponseWriter) Close() error {
	if c, ok := cw.writer().(io.WriteCloser); ok {
		return c.Close()
	}
	return nil
}

func (cw *compressResponseWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}

func encoderGzip(w io.Writer, level int) io.Writer {
	gw, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil
	}
	return gw
}

func encoderDeflate(w io.Writer, level int) io.Writer {
	dw, err := flate.NewWriter(w, level)
	if err != nil {
		return nil
	}
	return dw
}

// Compress creates a compression middleware using the full config
func Compress(cfg ...config.CompressConfig) func(http.Handler) http.Handler {
	c := config.DefaultCompressConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	validatePathConfig(c.ExcludedPaths, c.IncludedPaths, "Compress")

	compressor := NewCompressor(c.Level, c.Types...)
	compressor.excludedPaths = c.ExcludedPaths
	compressor.includedPaths = c.IncludedPaths

	// Set allowed algorithms and their precedence order
	compressor.algorithms = make(map[config.CompressionAlgorithm]bool)
	compressor.algorithmOrder = make([]config.CompressionAlgorithm, 0, len(c.Algorithms))

	// Process algorithms in order, registering encoders from providers or built-ins
	for _, alg := range c.Algorithms {
		compressor.algorithms[alg] = true
		algStr := strings.ToLower(string(alg))

		// Try to get encoder from providers first
		var encoder config.CompressionEncoder
		for _, provider := range c.Providers {
			if enc := provider.GetEncoder(algStr); enc != nil {
				encoder = enc
				break
			}
		}

		if encoder != nil {
			// Custom encoder from provider
			compressor.SetEncoder(encoder.Encoding(), func(w io.Writer, level int) io.Writer {
				return encoder.Encode(w, level)
			})
			compressor.algorithmOrder = append(compressor.algorithmOrder, alg)
		} else {
			// Built-in encoder
			switch alg {
			case config.Gzip:
				compressor.SetEncoder(httpx.ContentEncodingGzip, encoderGzip)
				compressor.algorithmOrder = append(compressor.algorithmOrder, alg)
			case config.Deflate:
				compressor.SetEncoder(httpx.ContentEncodingDeflate, encoderDeflate)
				compressor.algorithmOrder = append(compressor.algorithmOrder, alg)
			}
		}
	}

	return compressor.Handler
}

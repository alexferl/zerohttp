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
	exemptPaths        []string                             // Paths to skip compression
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
		exemptPaths:      []string{},
	}

	// Set default algorithms
	c.algorithms[config.Gzip] = true
	c.algorithms[config.Deflate] = true

	c.SetEncoder("deflate", encoderDeflate)
	c.SetEncoder("gzip", encoderGzip)
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

// isExemptPath checks if a path should be exempted from compression
func (c *Compressor) isExemptPath(path string) bool {
	for _, exemptPath := range c.exemptPaths {
		if pathMatches(path, exemptPath) {
			return true
		}
	}
	return false
}

// Handler returns a new middleware that will compress the response.
func (c *Compressor) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check exempt paths first
		if c.isExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		encoder, encoding, cleanup := c.selectEncoder(r.Header, w)
		cw := &compressResponseWriter{
			ResponseWriter:   w,
			w:                w,
			contentTypes:     c.allowedTypes,
			contentWildcards: c.allowedWildcards,
			encoding:         encoding,
			compressible:     false,
		}
		if encoder != nil {
			cw.w = encoder
		}

		defer cleanup()
		defer func() {
			_ = cw.Close()
		}()

		next.ServeHTTP(cw, r)
	})
}

// selectEncoder returns the encoder, the name of the encoder, and a closer function.
func (c *Compressor) selectEncoder(h http.Header, w io.Writer) (io.Writer, string, func()) {
	header := h.Get("Accept-Encoding")
	accepted := strings.Split(strings.ToLower(header), ",")

	for _, name := range c.encodingPrecedence {
		if matchAcceptEncoding(accepted, name) {
			// Check if algorithm is allowed
			var allowed bool
			switch name {
			case "gzip":
				allowed = c.algorithms[config.Gzip]
			case "deflate":
				allowed = c.algorithms[config.Deflate]
			default:
				allowed = true // Custom encoders are always allowed if added
			}

			if !allowed {
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
				return fn(w, c.level), name, func() {}
			}
		}
	}
	return nil, "", func() {}
}

func matchAcceptEncoding(accepted []string, encoding string) bool {
	for _, v := range accepted {
		v = strings.TrimSpace(v)
		if strings.Contains(v, encoding) || v == "*" {
			return true
		}
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
	buffer           []byte
}

func (cw *compressResponseWriter) isCompressible() bool {
	contentType := cw.Header().Get("Content-Type")
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
		cw.ResponseWriter.WriteHeader(code)
		return
	}
	cw.wroteHeader = true
	defer cw.ResponseWriter.WriteHeader(code)

	if cw.Header().Get("Content-Encoding") != "" {
		return
	}

	if !cw.isCompressible() {
		cw.compressible = false
		return
	}

	if cw.encoding != "" {
		cw.compressible = true
		cw.Header().Set("Content-Encoding", cw.encoding)
		cw.Header().Add("Vary", "Accept-Encoding")
		cw.Header().Del("Content-Length")
	}
}

func (cw *compressResponseWriter) Write(p []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
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
	// Flush any buffered data first
	if len(cw.buffer) > 0 {
		if cw.compressible {
			_, _ = cw.w.Write(cw.buffer)
		} else {
			_, _ = cw.ResponseWriter.Write(cw.buffer)
		}
		cw.buffer = nil
	}

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
	// Write any remaining buffered data
	if len(cw.buffer) > 0 {
		if cw.compressible {
			_, _ = cw.w.Write(cw.buffer)
		} else {
			_, _ = cw.ResponseWriter.Write(cw.buffer)
		}
		cw.buffer = nil
	}

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
func Compress(opts ...config.CompressOption) func(http.Handler) http.Handler {
	cfg := config.DefaultCompressConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Level <= 0 {
		cfg.Level = config.DefaultCompressConfig.Level
	}
	if cfg.Types == nil {
		cfg.Types = config.DefaultCompressConfig.Types
	}
	if cfg.Algorithms == nil {
		cfg.Algorithms = config.DefaultCompressConfig.Algorithms
	}
	if cfg.ExemptPaths == nil {
		cfg.ExemptPaths = config.DefaultCompressConfig.ExemptPaths
	}

	compressor := NewCompressor(cfg.Level, cfg.Types...)
	compressor.exemptPaths = cfg.ExemptPaths

	// Set allowed algorithms
	compressor.algorithms = make(map[config.CompressionAlgorithm]bool)
	for _, alg := range cfg.Algorithms {
		compressor.algorithms[alg] = true
	}

	return compressor.Handler
}

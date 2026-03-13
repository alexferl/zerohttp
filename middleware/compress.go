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

		reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

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
				encoder := fn(w, c.level)
				if encoder == nil {
					continue // Skip if encoder failed to create (invalid level)
				}
				return encoder, name, func() {}
			}
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
			if q == "0" || strings.HasPrefix(q, "0.0") {
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
		// Ignore subsequent WriteHeader calls, matching standard library behavior
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
		c = cfg[0]
	}

	if c.Level <= 0 {
		c.Level = config.DefaultCompressConfig.Level
	}
	if c.Types == nil {
		c.Types = config.DefaultCompressConfig.Types
	}
	if c.Algorithms == nil {
		c.Algorithms = config.DefaultCompressConfig.Algorithms
	}
	if c.ExemptPaths == nil {
		c.ExemptPaths = config.DefaultCompressConfig.ExemptPaths
	}

	compressor := NewCompressor(c.Level, c.Types...)
	compressor.exemptPaths = c.ExemptPaths

	// Set allowed algorithms
	compressor.algorithms = make(map[config.CompressionAlgorithm]bool)
	for _, alg := range c.Algorithms {
		compressor.algorithms[alg] = true
	}

	// Add custom encoders from provider if set
	if c.Provider != nil {
		for _, alg := range c.Algorithms {
			if encoder := c.Provider.GetEncoder(string(alg)); encoder != nil {
				compressor.SetEncoder(encoder.Encoding(), func(w io.Writer, level int) io.Writer {
					return encoder.Encode(w, level)
				})
			}
		}
	}

	return compressor.Handler
}

package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/alexferl/zerohttp/config"
)

// compressResponseWriter wraps http.ResponseWriter to handle compression
type compressResponseWriter struct {
	http.ResponseWriter
	compressor    io.WriteCloser
	algorithm     config.CompressionAlgorithm
	minSize       int
	types         map[string]bool
	buffer        []byte
	headerWritten bool
	compressed    bool
}

func (w *compressResponseWriter) WriteHeader(code int) {
	if w.headerWritten {
		return
	}
	w.headerWritten = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *compressResponseWriter) Write(data []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	if !w.compressed && w.compressor == nil {
		w.buffer = append(w.buffer, data...)

		if len(w.buffer) >= w.minSize {
			w.startCompressionIfNeeded()

			if w.compressed {
				w.compressor = w.createCompressor()
				if w.compressor == nil {
					w.compressed = false
				}
			}
		}

		if w.compressed && w.compressor != nil {
			return w.flushBufferToCompressor()
		}

		return len(data), nil
	}

	if w.compressed && w.compressor != nil {
		return w.compressor.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

func (w *compressResponseWriter) createCompressor() io.WriteCloser {
	switch w.algorithm {
	case config.Gzip:
		compressor, err := gzip.NewWriterLevel(w.ResponseWriter, 6)
		if err != nil {
			return nil
		}
		return compressor
	case config.Deflate:
		compressor, err := flate.NewWriter(w.ResponseWriter, 6)
		if err != nil {
			return nil
		}
		return compressor
	default:
		return nil
	}
}

func (w *compressResponseWriter) startCompressionIfNeeded() {
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(w.buffer)
		w.Header().Set("Content-Type", contentType)
	}

	shouldCompress := false
	for mimeType := range w.types {
		if strings.HasPrefix(strings.ToLower(contentType), mimeType) {
			shouldCompress = true
			break
		}
	}

	if shouldCompress && len(w.buffer) >= w.minSize {
		w.compressed = true
		w.Header().Set("Content-Encoding", string(w.algorithm))
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length") // Let the compressor set this
	}
}

func (w *compressResponseWriter) flushBufferToCompressor() (int, error) {
	totalWritten := len(w.buffer)
	if w.compressor != nil {
		_, err := w.compressor.Write(w.buffer)
		if err != nil {
			return totalWritten, err
		}
	}
	w.buffer = nil
	return totalWritten, nil
}

func (w *compressResponseWriter) Close() error {
	// Flush any remaining buffer
	if w.buffer != nil && !w.compressed {
		// Not compressed, write directly to response
		_, err := w.ResponseWriter.Write(w.buffer)
		if err != nil {
			return err
		}
		w.buffer = nil
	}

	if w.compressor != nil {
		if err := w.compressor.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Compress creates a compression middleware using standard library algorithms
func Compress(opts ...config.CompressOption) func(http.Handler) http.Handler {
	cfg := config.DefaultCompressConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Level <= 0 {
		cfg.Level = config.DefaultCompressConfig.Level
	}
	if cfg.MinSize <= 0 {
		cfg.MinSize = config.DefaultCompressConfig.MinSize
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

	typeMap := make(map[string]bool)
	for _, t := range cfg.Types {
		typeMap[strings.ToLower(t)] = true
	}

	algorithmMap := make(map[config.CompressionAlgorithm]bool)
	for _, alg := range cfg.Algorithms {
		algorithmMap[alg] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			acceptEncoding := r.Header.Get("Accept-Encoding")
			if acceptEncoding == "" {
				next.ServeHTTP(w, r)
				return
			}

			var selectedAlgorithm config.CompressionAlgorithm

			encodings := strings.Split(acceptEncoding, ",")
			for _, encoding := range encodings {
				encoding = strings.TrimSpace(strings.ToLower(encoding))

				if (strings.HasPrefix(encoding, "gzip") || strings.HasPrefix(encoding, "*")) && algorithmMap[config.Gzip] {
					selectedAlgorithm = config.Gzip
					break
				} else if strings.HasPrefix(encoding, "deflate") && algorithmMap[config.Deflate] {
					selectedAlgorithm = config.Deflate
					break
				}
			}

			if selectedAlgorithm == "" {
				next.ServeHTTP(w, r)
				return
			}

			cw := &compressResponseWriter{
				ResponseWriter: w,
				algorithm:      selectedAlgorithm,
				minSize:        cfg.MinSize,
				types:          typeMap,
				buffer:         make([]byte, 0, cfg.MinSize),
			}

			defer func() {
				_ = cw.Close()
			}()

			next.ServeHTTP(cw, r)
		})
	}
}

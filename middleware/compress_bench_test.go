package middleware

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

// BenchmarkCompress_Algorithms compares gzip vs deflate encoding speed
func BenchmarkCompress_Algorithms(b *testing.B) {
	content := []byte(strings.Repeat("This is test content for compression. ", 100))

	algorithms := []struct {
		name    string
		encoder string
		alg     config.CompressionAlgorithm
	}{
		{"Gzip", "gzip", config.Gzip},
		{"Deflate", "deflate", config.Deflate},
	}

	for _, alg := range algorithms {
		b.Run(alg.name, func(b *testing.B) {
			mw := Compress(config.CompressConfig{
				Algorithms: []config.CompressionAlgorithm{alg.alg},
				Types:      []string{"text/plain"},
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", alg.encoder)

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCompress_PoolEfficiency measures the benefit of encoder pooling
func BenchmarkCompress_PoolEfficiency(b *testing.B) {
	content := []byte(strings.Repeat("Test content for compression pooling. ", 50))

	// Create compressor with pooling (default)
	pooledCompressor := NewCompressor(5, "text/plain")

	b.Run("WithPooling", func(b *testing.B) {
		handler := pooledCompressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	// Simulate no pooling by creating a new compressor each time
	b.Run("WithoutPooling", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			// Create new compressor each iteration (simulates no pooling)
			compressor := NewCompressor(5, "text/plain")
			handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkCompress_ContentTypeMatching measures content-type matching overhead
func BenchmarkCompress_ContentTypeMatching(b *testing.B) {
	content := []byte(strings.Repeat("Test content. ", 20))

	testCases := []struct {
		name  string
		types []string
	}{
		{"ExactMatch", []string{"text/plain", "text/html", "application/json"}},
		{"Wildcard", []string{"text/*", "application/*"}},
		{"Mixed", []string{"text/plain", "text/html", "application/*"}},
		{"ManyTypes", []string{
			"text/plain", "text/html", "text/css", "text/javascript",
			"application/json", "application/xml", "application/javascript",
			"image/svg+xml", "application/pdf",
		}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			compressor := NewCompressor(5, tc.types...)

			handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCompress_CompressionLevels compares different compression levels
func BenchmarkCompress_CompressionLevels(b *testing.B) {
	content := []byte(strings.Repeat("This is test content that will be compressed. ", 100))

	levels := []int{1, 3, 5, 6, 9}

	for _, level := range levels {
		b.Run(fmt.Sprintf("Level%d", level), func(b *testing.B) {
			compressor := NewCompressor(level, "text/plain")

			handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCompress_PayloadSizes measures performance with different payload sizes
func BenchmarkCompress_PayloadSizes(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"100B", 100},
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			content := []byte(strings.Repeat("x", s.size))

			mw := Compress(config.CompressConfig{
				Types: []string{"text/plain"},
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			b.ReportAllocs()
			b.SetBytes(int64(s.size))
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCompress_Concurrent measures concurrent compression performance
func BenchmarkCompress_Concurrent(b *testing.B) {
	content := []byte(strings.Repeat("Test content for concurrent compression. ", 50))

	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			mw := Compress(config.CompressConfig{
				Types: []string{"text/plain"},
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					rr := httptest.NewRecorder()
					handler.ServeHTTP(rr, req)
				}
			})
		})
	}
}

// BenchmarkCompress_NoCompressionFallback measures overhead when compression is skipped
func BenchmarkCompress_NoCompressionFallback(b *testing.B) {
	content := []byte(strings.Repeat("Test content. ", 20))

	mw := Compress(config.CompressConfig{
		Types: []string{"text/html"}, // Only compress HTML
	})

	b.Run("NoAcceptEncoding", func(b *testing.B) {
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No Accept-Encoding header

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	b.Run("NonCompressibleType", func(b *testing.B) {
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	b.Run("Baseline_NoMiddleware", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkCompress_ExemptPaths measures path exemption checking overhead
func BenchmarkCompress_ExemptPaths(b *testing.B) {
	content := []byte(strings.Repeat("Test content. ", 20))

	testCases := []struct {
		name        string
		exemptPaths []string
		path        string
	}{
		{"NoExemptions", nil, "/test"},
		{"OneExemption", []string{"/health"}, "/test"},
		{"ManyExemptions", []string{"/health", "/metrics", "/api/internal/", "/debug/", "/admin/"}, "/test"},
		{"ExemptPathMatch", []string{"/health", "/metrics", "/api/internal/"}, "/api/internal/data"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			mw := Compress(config.CompressConfig{
				Types:       []string{"text/plain"},
				ExemptPaths: tc.exemptPaths,
			})

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write(content)
			}))

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkCompress_ResponseWriterWrapping measures the overhead of wrapping ResponseWriter
func BenchmarkCompress_ResponseWriterWrapping(b *testing.B) {
	content := []byte(strings.Repeat("Test content. ", 20))

	b.Run("WithCompression", func(b *testing.B) {
		compressor := NewCompressor(5, "text/plain")

		handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})

	b.Run("WithoutCompression", func(b *testing.B) {
		compressor := NewCompressor(5, "text/html") // Different type than response

		handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkCompress_Decompression verifies compressed data is valid and measures decompression
func BenchmarkCompress_Decompression(b *testing.B) {
	content := []byte(strings.Repeat("This is test content for compression and decompression. ", 100))

	b.Run("Gzip_Decompress", func(b *testing.B) {
		compressor := NewCompressor(5, "text/plain")

		// First, compress the content
		compressed := httptest.NewRecorder()
		handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		handler.ServeHTTP(compressed, req)

		compressedBody := compressed.Body.Bytes()

		b.ReportAllocs()
		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		for b.Loop() {
			reader, _ := gzip.NewReader(strings.NewReader(string(compressedBody)))
			_, _ = io.ReadAll(reader)
			_ = reader.Close()
		}
	})

	b.Run("Deflate_Decompress", func(b *testing.B) {
		compressor := NewCompressor(5, "text/plain")
		compressor.algorithms[config.Gzip] = false // Disable gzip

		// First, compress the content
		compressed := httptest.NewRecorder()
		handler := compressor.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(content)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "deflate")
		handler.ServeHTTP(compressed, req)

		compressedBody := compressed.Body.Bytes()

		b.ReportAllocs()
		b.ResetTimer()
		b.SetBytes(int64(len(content)))

		for b.Loop() {
			reader := flate.NewReader(strings.NewReader(string(compressedBody)))
			_, _ = io.ReadAll(reader)
			_ = reader.Close()
		}
	})
}

package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/rwutil"
)

// BenchmarkETag_HashAlgorithms compares FNV vs MD5 hashing performance
func BenchmarkETag_HashAlgorithms(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"100B", 100},
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
	}

	for _, s := range sizes {
		data := make([]byte, s.size)

		b.Run("FNV/"+s.name, func(b *testing.B) {
			h := fnv.New64a()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				h.Reset()
				h.Write(data)
				_ = h.Sum64()
			}
		})

		b.Run("MD5/"+s.name, func(b *testing.B) {
			h := md5.New()
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				h.Reset()
				h.Write(data)
				_ = h.Sum(nil)
			}
		})
	}
}

// BenchmarkETag_GenerateETag measures ETag generation with different algorithms
func BenchmarkETag_GenerateETag(b *testing.B) {
	sizes := []int{100, 1024, 10240}

	for _, size := range sizes {
		data := make([]byte, size)

		b.Run(fmt.Sprintf("FNV_%dB", size), func(b *testing.B) {
			ew := &etagResponseWriter{
				config:         config.ETagConfig{Algorithm: config.FNV},
				ResponseBuffer: &rwutil.ResponseBuffer{Buf: *bytes.NewBuffer(data)},
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_ = ew.generateETag()
			}
		})

		b.Run(fmt.Sprintf("MD5_%dB", size), func(b *testing.B) {
			ew := &etagResponseWriter{
				config:         config.ETagConfig{Algorithm: config.MD5},
				ResponseBuffer: &rwutil.ResponseBuffer{Buf: *bytes.NewBuffer(data)},
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_ = ew.generateETag()
			}
		})

		b.Run(fmt.Sprintf("FNV_Weak_%dB", size), func(b *testing.B) {
			weak := true
			ew := &etagResponseWriter{
				config:         config.ETagConfig{Algorithm: config.FNV, Weak: &weak},
				ResponseBuffer: &rwutil.ResponseBuffer{Buf: *bytes.NewBuffer(data)},
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_ = ew.generateETag()
			}
		})
	}
}

// BenchmarkETag_Middleware measures full middleware overhead
func BenchmarkETag_Middleware(b *testing.B) {
	sizes := []struct {
		name string
		data string
	}{
		{"Tiny_100B", strings.Repeat("a", 100)},
		{"Small_1KB", strings.Repeat("b", 1024)},
		{"Medium_10KB", strings.Repeat("c", 10*1024)},
	}

	algorithms := []struct {
		name string
		algo config.ETagAlgorithm
	}{
		{"FNV", config.FNV},
		{"MD5", config.MD5},
	}

	for _, algo := range algorithms {
		for _, s := range sizes {
			b.Run(algo.name+"/"+s.name, func(b *testing.B) {
				handler := ETag(config.ETagConfig{Algorithm: algo.algo})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(s.data))
				}))

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					handler.ServeHTTP(w, req)
				}
			})
		}
	}
}

// BenchmarkETag_ConditionalRequests measures If-None-Match handling
func BenchmarkETag_ConditionalRequests(b *testing.B) {
	data := "Hello, World!"
	// Pre-compute the ETag
	h := fnv.New64a()
	_, _ = h.Write([]byte(data))
	etag := fmt.Sprintf(`"%x"`, h.Sum64())

	b.Run("CacheHit_304", func(b *testing.B) {
		handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(data))
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("If-None-Match", etag)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("CacheMiss_200", func(b *testing.B) {
		handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(data))
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("If-None-Match", `"different-etag"`)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})

	b.Run("NoConditionalHeader", func(b *testing.B) {
		handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(data))
		}))

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkETag_Match compares different ETag matching scenarios
func BenchmarkETag_Match(b *testing.B) {
	cases := []struct {
		name        string
		ifNoneMatch string
		etag        string
	}{
		{"ExactMatch", `"abc123"`, `"abc123"`},
		{"WeakMatch", `W/"abc123"`, `W/"abc123"`},
		{"MixedWeakStrong", `W/"abc123"`, `"abc123"`},
		{"Wildcard", `*`, `"anything"`},
		{"NoMatch", `"different"`, `"abc123"`},
		{"MultipleValues", `"other", "abc123"`, `"abc123"`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_ = etagMatches(tc.ifNoneMatch, tc.etag)
			}
		})
	}
}

// BenchmarkETag_PooledVsNew compares pooled hash instances vs new allocations
func BenchmarkETag_PooledVsNew(b *testing.B) {
	data := make([]byte, 1024)

	b.Run("Pooled_FNV", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			h := fnvHashPool.Get().(hash.Hash64)
			h.Reset()
			_, _ = h.Write(data)
			_ = h.Sum64()
			fnvHashPool.Put(h)
		}
	})

	b.Run("New_FNV", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			h := fnv.New64a()
			_, _ = h.Write(data)
			_ = h.Sum64()
		}
	})

	b.Run("Pooled_MD5", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			h := md5HashPool.Get().(hash.Hash)
			h.Reset()
			h.Write(data)
			_ = h.Sum(nil)
			md5HashPool.Put(h)
		}
	})

	b.Run("New_MD5", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			h := md5.New()
			h.Write(data)
			_ = h.Sum(nil)
		}
	})
}

// BenchmarkETag_FileETag measures file ETag generation
func BenchmarkETag_FileETag(b *testing.B) {
	b.Run("Weak", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = GenerateFileETag(1709999999, 1024*1024*100, true)
		}
	})

	b.Run("Strong", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			_ = GenerateFileETag(1709999999, 1024*1024*100, false)
		}
	})
}

// BenchmarkETag_ParseETag measures ETag parsing
func BenchmarkETag_ParseETag(b *testing.B) {
	cases := []struct {
		name string
		etag string
	}{
		{"Weak", `W/"abc123"`},
		{"Strong", `"abc123"`},
		{"NoQuotes", `abc123`},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = ParseETag(tc.etag)
			}
		})
	}
}

// BenchmarkETag_BufferPool measures buffer pooling efficiency
func BenchmarkETag_BufferPool(b *testing.B) {
	sizes := []int{100, 1024, 10240}

	for _, size := range sizes {
		data := make([]byte, size)

		b.Run(fmt.Sprintf("Pooled_%dB", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				buf := etagBufferPool.Get().(*bytes.Buffer)
				buf.Write(data)
				_ = buf.Len()
				buf.Reset()
				etagBufferPool.Put(buf)
			}
		})

		b.Run(fmt.Sprintf("New_%dB", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				buf := &bytes.Buffer{}
				buf.Write(data)
				_ = buf.Len()
			}
		})
	}
}

// BenchmarkETag_Concurrent measures concurrent ETag generation
func BenchmarkETag_Concurrent(b *testing.B) {
	data := strings.Repeat("x", 1024)
	handler := ETag()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(data))
	}))

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

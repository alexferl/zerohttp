package zerohttp

import (
	"embed"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

//go:embed testdata/static
var benchStaticFS embed.FS

//go:embed testdata/files
var benchFilesFS embed.FS

// BenchmarkStatic_EmbeddedFSVsDirectory compares serving static files
// from embedded FS vs directory serving.
func BenchmarkStatic_EmbeddedFSVsDirectory(b *testing.B) {
	b.Run("EmbeddedFS", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", false)

		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Directory", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.StaticDir("testdata/static", false)

		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkStatic_FallbackBehavior measures the overhead of SPA mode
// fallback (serving index.html for non-existent files).
func BenchmarkStatic_FallbackBehavior(b *testing.B) {
	b.Run("WithoutFallback_ExistingFile", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", false)

		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("WithFallback_ExistingFile", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", true)

		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("WithFallback_FallbackToIndex", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", true)

		// Non-existent path that triggers fallback to index.html
		req := httptest.NewRequest(http.MethodGet, "/some/client/route", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("WithoutFallback_FileNotFound", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", false)

		// Non-existent path that returns 404
		req := httptest.NewRequest(http.MethodGet, "/nonexistent/file", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkStatic_APIPrefixMatching measures the overhead of API prefix
// checking for exclusions (e.g., /api/ routes).
func BenchmarkStatic_APIPrefixMatching(b *testing.B) {
	testCases := []struct {
		name        string
		apiPrefixes []string
		path        string
	}{
		{"NoPrefixes", nil, "/app.js"},
		{"OnePrefix_NoMatch", []string{"/api/"}, "/app.js"},
		{"OnePrefix_Match", []string{"/api/"}, "/api/users"},
		{"ManyPrefixes_NoMatch", []string{"/api/", "/v1/", "/v2/", "/internal/", "/admin/"}, "/app.js"},
		{"ManyPrefixes_FirstMatch", []string{"/api/", "/v1/", "/v2/", "/internal/", "/admin/"}, "/api/users"},
		{"ManyPrefixes_LastMatch", []string{"/api/", "/v1/", "/v2/", "/internal/", "/admin/"}, "/admin/users"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			router := NewRouter()
			router.SetLogger(&noopLogger{})
			router.Static(benchStaticFS, "testdata/static", true, tc.apiPrefixes...)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkStatic_FileNotFoundHandling measures 404 response handling
// for different file serving methods.
func BenchmarkStatic_FileNotFoundHandling(b *testing.B) {
	b.Run("EmbeddedFS_Static", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", false)

		req := httptest.NewRequest(http.MethodGet, "/nonexistent.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Directory_StaticDir", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.StaticDir("testdata/static", false)

		req := httptest.NewRequest(http.MethodGet, "/nonexistent.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("EmbeddedFS_Files", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Files("/static/", benchFilesFS, "testdata/files")

		req := httptest.NewRequest(http.MethodGet, "/static/nonexistent.txt", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Directory_FilesDir", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.FilesDir("/files/", "testdata/files")

		req := httptest.NewRequest(http.MethodGet, "/files/nonexistent.txt", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkStatic_ETagGeneration measures ETag generation overhead when
// serving static files.
func BenchmarkStatic_ETagGeneration(b *testing.B) {
	b.Run("WithETag", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		cfg := DefaultConfig
		cfg.RequestLogger.LogRequestBody = false
		cfg.RequestLogger.LogResponseBody = false
		router.SetConfig(cfg)
		router.Static(benchStaticFS, "testdata/static", false)

		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("ConditionalRequest", func(b *testing.B) {
		router := NewRouter()
		router.SetLogger(&noopLogger{})
		router.Static(benchStaticFS, "testdata/static", false)

		// First request to get the ETag
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		etag := w.Header().Get(httpx.HeaderETag)

		// Now benchmark conditional requests with If-None-Match
		req = httptest.NewRequest(http.MethodGet, "/app.js", nil)
		req.Header.Set(httpx.HeaderIfNoneMatch, etag)

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkStatic_FileTypes measures serving different file types.
func BenchmarkStatic_FileTypes(b *testing.B) {
	// Create temporary files of different types for benchmarking
	tmpDir := b.TempDir()

	files := map[string]struct {
		name    string
		content []byte
	}{
		"SmallJS":   {"small.js", []byte("console.log('hello');")},
		"SmallHTML": {"small.html", []byte("<!DOCTYPE html><html><body>Hi</body></html>")},
		"LargeJS":   {"large.js", make([]byte, 50000)},
		"LargeHTML": {"large.html", make([]byte, 50000)},
	}

	// Fill large files with content
	for i := range files["LargeJS"].content {
		files["LargeJS"].content[i] = byte('a' + i%26)
	}
	for i := range files["LargeHTML"].content {
		files["LargeHTML"].content[i] = byte('a' + i%26)
	}

	for name, file := range files {
		path := filepath.Join(tmpDir, file.name)
		if err := os.WriteFile(path, file.content, 0o644); err != nil {
			b.Fatalf("failed to create test file: %v", err)
		}

		b.Run(name, func(b *testing.B) {
			router := NewRouter()
			router.SetLogger(&noopLogger{})
			router.FilesDir("/files/", tmpDir)

			req := httptest.NewRequest(http.MethodGet, "/files/"+file.name, nil)
			b.SetBytes(int64(len(file.content)))

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkStatic_Concurrent measures concurrent static file serving performance.
func BenchmarkStatic_Concurrent(b *testing.B) {
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Goroutines%d", concurrency), func(b *testing.B) {
			router := NewRouter()
			router.SetLogger(&noopLogger{})
			router.Static(benchStaticFS, "testdata/static", false)

			req := httptest.NewRequest(http.MethodGet, "/app.js", nil)

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					w := httptest.NewRecorder()
					router.ServeHTTP(w, req)
				}
			})
		})
	}
}

// BenchmarkStatic_DirectoryTraversal measures overhead of path cleaning
// and directory traversal prevention.
func BenchmarkStatic_DirectoryTraversal(b *testing.B) {
	testPaths := []struct {
		name string
		path string
	}{
		{"CleanPath", "/app.js"},
		{"WithDotSlash", "/./app.js"},
		{"WithDoubleSlash", "//app.js"},
		{"WithParentDir", "/../../app.js"},
		{"DeepNested", "/a/b/c/../../../app.js"},
	}

	for _, tc := range testPaths {
		b.Run(tc.name, func(b *testing.B) {
			router := NewRouter()
			router.SetLogger(&noopLogger{})
			router.Static(benchStaticFS, "testdata/static", false)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
			}
		})
	}
}

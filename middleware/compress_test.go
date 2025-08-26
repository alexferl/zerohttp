package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestCompress_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name             string
		acceptEncoding   string
		contentType      string
		content          string
		minSize          int
		expectedEncoding string
		shouldCompress   bool
	}{
		{
			name:             "gzip compression large content",
			acceptEncoding:   "gzip",
			contentType:      "text/plain",
			content:          strings.Repeat("Hello World! ", 20),
			minSize:          10,
			expectedEncoding: "gzip",
			shouldCompress:   true,
		},
		{
			name:             "deflate compression",
			acceptEncoding:   "deflate",
			contentType:      "text/plain",
			content:          strings.Repeat("Hello World! ", 10),
			minSize:          10,
			expectedEncoding: "deflate",
			shouldCompress:   true,
		},
		{
			name:             "wildcard accepts gzip",
			acceptEncoding:   "*",
			contentType:      "text/plain",
			content:          strings.Repeat("test ", 10),
			minSize:          10,
			expectedEncoding: "gzip",
			shouldCompress:   true,
		},
		{
			name:             "gzip priority over deflate",
			acceptEncoding:   "gzip, deflate",
			contentType:      "text/plain",
			content:          strings.Repeat("test ", 10),
			minSize:          10,
			expectedEncoding: "gzip",
			shouldCompress:   true,
		},
		{
			name:             "small content not compressed",
			acceptEncoding:   "gzip",
			contentType:      "application/json",
			content:          "small",
			minSize:          1024,
			expectedEncoding: "",
			shouldCompress:   false,
		},
		{
			name:             "no accept encoding",
			acceptEncoding:   "",
			contentType:      "text/plain",
			content:          strings.Repeat("test ", 10),
			minSize:          10,
			expectedEncoding: "",
			shouldCompress:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Compress(config.WithCompressMinSize(tt.minSize))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			rr := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				_, err := w.Write([]byte(tt.content))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
			middleware(next).ServeHTTP(rr, req)

			actualEncoding := rr.Header().Get("Content-Encoding")
			if actualEncoding != tt.expectedEncoding {
				t.Errorf("Expected Content-Encoding %s, got %s", tt.expectedEncoding, actualEncoding)
			}

			if tt.shouldCompress {
				if rr.Header().Get("Vary") != "Accept-Encoding" {
					t.Error("Expected Vary header to be set")
				}

				if rr.Header().Get("Content-Length") != "" {
					t.Error("Content-Length should be deleted when compressing")
				}

				var decompressed []byte
				var err error

				switch tt.expectedEncoding {
				case "gzip":
					reader, err := gzip.NewReader(rr.Body)
					if err != nil {
						t.Fatalf("Failed to create gzip reader: %v", err)
					}
					if err := reader.Close(); err != nil {
						t.Fatalf("failed to close reader: %v", err)
					}
					decompressed, err = io.ReadAll(reader)
					if err != nil {
						t.Fatalf("Failed to read decompressed data: %v", err)
					}
				case "deflate":
					reader := flate.NewReader(rr.Body)
					if err := reader.Close(); err != nil {
						t.Fatalf("failed to close reader: %v", err)
					}
					decompressed, err = io.ReadAll(reader)
					if err != nil {
						t.Fatalf("Failed to read decompressed data: %v", err)
					}
				}

				if err != nil {
					t.Fatalf("Failed to decompress: %v", err)
				}

				if string(decompressed) != tt.content {
					t.Errorf("Decompressed content doesn't match original")
				}
			} else {
				if rr.Body.String() != tt.content {
					t.Error("Uncompressed content should be passed through unmodified")
				}
			}
		})
	}
}

func TestCompress_ExemptPaths(t *testing.T) {
	middleware := Compress(
		config.WithCompressMinSize(10),
		config.WithCompressExemptPaths([]string{"/health", "/metrics", "/api/internal/"}),
	)

	tests := []struct {
		path           string
		shouldCompress bool
	}{
		{"/health", false},
		{"/metrics", false},
		{"/api/internal/", false},
		{"/api/internal/status", false},
		{"/api/public", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()
			largeContent := strings.Repeat("test content ", 10)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, err := w.Write([]byte(largeContent))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
			middleware(next).ServeHTTP(rr, req)

			hasCompression := rr.Header().Get("Content-Encoding") != ""
			if hasCompression != tt.shouldCompress {
				t.Errorf("Expected compression=%v, got compression=%v", tt.shouldCompress, hasCompression)
			}
		})
	}
}

func TestCompress_ContentTypes(t *testing.T) {
	middleware := Compress(
		config.WithCompressMinSize(10),
		config.WithCompressTypes([]string{"text/plain", "application/json"}),
	)

	tests := []struct {
		name           string
		contentType    string
		autoDetect     bool
		shouldCompress bool
	}{
		{"supported text/plain", "text/plain", false, true},
		{"supported application/json", "application/json", false, true},
		{"unsupported application/octet-stream", "application/octet-stream", false, false},
		{"auto-detect text", "", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rr := httptest.NewRecorder()
			largeContent := strings.Repeat("test content ", 10)
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !tt.autoDetect {
					w.Header().Set("Content-Type", tt.contentType)
				}
				_, err := w.Write([]byte(largeContent))
				if err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			})
			middleware(next).ServeHTTP(rr, req)

			hasCompression := rr.Header().Get("Content-Encoding") != ""
			if hasCompression != tt.shouldCompress {
				t.Errorf("Expected compression=%v, got compression=%v", tt.shouldCompress, hasCompression)
			}
			if tt.autoDetect && rr.Header().Get("Content-Type") == "" {
				t.Error("Content-Type should be auto-detected")
			}
		})
	}
}

func TestCompress_MultipleWrites(t *testing.T) {
	middleware := Compress(config.WithCompressMinSize(5))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte("Hello ")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
		if _, err := w.Write([]byte("World! ")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
		if _, err := w.Write([]byte("This is a test.")); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware(next).ServeHTTP(rr, req)

	expectedContent := "Hello World! This is a test."
	if rr.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected gzip compression for multiple writes")
	}

	reader, err := gzip.NewReader(rr.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("failed to close gzip reader: %v", err)
	}

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read decompressed content: %v", err)
	}
	if string(decompressed) != expectedContent {
		t.Errorf("Decompressed content doesn't match expected")
	}
}

func TestCompress_EdgeCases(t *testing.T) {
	t.Run("multiple write header calls", func(t *testing.T) {
		middleware := Compress(config.WithCompressMinSize(10))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rr := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.WriteHeader(http.StatusCreated)
			_, err := w.Write([]byte(strings.Repeat("test ", 10)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})
		middleware(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("unsupported algorithm", func(t *testing.T) {
		middleware := Compress(
			config.WithCompressMinSize(10),
			config.WithCompressAlgorithms([]config.CompressionAlgorithm{"unsupported"}),
		)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Encoding", "unsupported")
		rr := httptest.NewRecorder()
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, err := w.Write([]byte(strings.Repeat("test ", 10)))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		})
		middleware(next).ServeHTTP(rr, req)

		if rr.Header().Get("Content-Encoding") != "" {
			t.Error("Should not compress with unsupported algorithm")
		}
	})
}

func TestCompress_ConfigFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		config   func() func(http.Handler) http.Handler
		testFunc func(t *testing.T, middleware func(http.Handler) http.Handler)
	}{
		{
			name: "level fallback",
			config: func() func(http.Handler) http.Handler {
				return Compress(
					config.WithCompressLevel(0),
					config.WithCompressMinSize(10),
				)
			},
			testFunc: func(t *testing.T, middleware func(http.Handler) http.Handler) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rr := httptest.NewRecorder()
				middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					_, err := w.Write([]byte(strings.Repeat("test ", 10)))
					if err != nil {
						t.Fatalf("failed to write response: %v", err)
					}
				})).ServeHTTP(rr, req)

				if rr.Header().Get("Content-Encoding") != "gzip" {
					t.Error("Should use default level when 0 provided")
				}
			},
		},
		{
			name: "min size fallback",
			config: func() func(http.Handler) http.Handler {
				return Compress(config.WithCompressMinSize(0))
			},
			testFunc: func(t *testing.T, middleware func(http.Handler) http.Handler) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rr := httptest.NewRecorder()
				middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					_, err := w.Write([]byte("small"))
					if err != nil {
						t.Fatalf("failed to write response: %v", err)
					}
				})).ServeHTTP(rr, req)

				if rr.Header().Get("Content-Encoding") != "" {
					t.Error("Should use default minSize when 0 provided")
				}
			},
		},
		{
			name: "types fallback",
			config: func() func(http.Handler) http.Handler {
				return Compress(
					config.WithCompressTypes(nil),
					config.WithCompressMinSize(10),
				)
			},
			testFunc: func(t *testing.T, middleware func(http.Handler) http.Handler) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rr := httptest.NewRecorder()
				middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					_, err := w.Write([]byte(strings.Repeat("test ", 10)))
					if err != nil {
						t.Fatalf("failed to write response: %v", err)
					}
				})).ServeHTTP(rr, req)

				if rr.Header().Get("Content-Encoding") != "gzip" {
					t.Error("Should use default types when nil provided")
				}
			},
		},
		{
			name: "algorithms fallback",
			config: func() func(http.Handler) http.Handler {
				return Compress(
					config.WithCompressAlgorithms(nil),
					config.WithCompressMinSize(10),
				)
			},
			testFunc: func(t *testing.T, middleware func(http.Handler) http.Handler) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rr := httptest.NewRecorder()
				middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					_, err := w.Write([]byte(strings.Repeat("test ", 10)))
					if err != nil {
						t.Fatalf("failed to write response: %v", err)
					}
				})).ServeHTTP(rr, req)

				if rr.Header().Get("Content-Encoding") != "gzip" {
					t.Error("Should use default algorithms when nil provided")
				}
			},
		},
		{
			name: "exempt paths fallback",
			config: func() func(http.Handler) http.Handler {
				return Compress(
					config.WithCompressExemptPaths(nil),
					config.WithCompressMinSize(10),
				)
			},
			testFunc: func(t *testing.T, middleware func(http.Handler) http.Handler) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Accept-Encoding", "gzip")
				rr := httptest.NewRecorder()
				middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/plain")
					_, err := w.Write([]byte(strings.Repeat("test ", 10)))
					if err != nil {
						t.Fatalf("failed to write response: %v", err)
					}
				})).ServeHTTP(rr, req)

				if rr.Header().Get("Content-Encoding") != "gzip" {
					t.Error("Should use default exempt paths when nil provided")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := tt.config()
			tt.testFunc(t, middleware)
		})
	}
}

package contentencoding

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestContentEncodingConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 2, len(cfg.Encodings))
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	expectedEncodings := []string{"gzip", "deflate"}
	zhtest.AssertDeepEqual(t, expectedEncodings, cfg.Encodings)
	zhtest.AssertEqual(t, "gzip", cfg.Encodings[0])
	zhtest.AssertEqual(t, "deflate", cfg.Encodings[1])
}

func TestContentEncodingConfig_StructAssignment(t *testing.T) {
	t.Run("encodings assignment", func(t *testing.T) {
		tests := []struct {
			name     string
			input    []string
			expected []string
		}{
			{"multiple encodings", []string{"gzip", "deflate", "br", "compress"}, []string{"gzip", "deflate", "br", "compress"}},
			{"brotli only", []string{"br"}, []string{"br"}},
			{"all common encodings", []string{"gzip", "deflate", "br", "compress", "identity"}, []string{"gzip", "deflate", "br", "compress", "identity"}},
			{"custom encodings", []string{"lz4", "zstd", "snappy", "lzma"}, []string{"lz4", "zstd", "snappy", "lzma"}},
			{"identity encoding", []string{"identity", "gzip"}, []string{"identity", "gzip"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := Config{
					Encodings: tt.input,
				}
				zhtest.AssertEqual(t, len(tt.expected), len(cfg.Encodings))
				zhtest.AssertDeepEqual(t, tt.expected, cfg.Encodings)
			})
		}
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/upload", "/health", "/metrics", "/static"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
	})

	t.Run("single encoding variants", func(t *testing.T) {
		testCases := []string{"gzip", "deflate", "br", "compress", "identity"}
		for _, encoding := range testCases {
			cfg := Config{
				Encodings: []string{encoding},
			}
			zhtest.AssertEqual(t, 1, len(cfg.Encodings))
			zhtest.AssertEqual(t, encoding, cfg.Encodings[0])
		}
	})
}

func TestContentEncodingConfig_MultipleFields(t *testing.T) {
	encodings := []string{"br", "gzip"}
	excludedPaths := []string{"/upload", "/download"}
	includedPaths := []string{"/api/public"}
	cfg := Config{
		Encodings:     encodings,
		ExcludedPaths: excludedPaths,
		IncludedPaths: includedPaths,
	}

	zhtest.AssertEqual(t, 2, len(cfg.Encodings))
	zhtest.AssertDeepEqual(t, encodings, cfg.Encodings)
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
	zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
}

func TestContentEncodingConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := Config{
			Encodings:     []string{},
			ExcludedPaths: []string{},
			IncludedPaths: []string{},
		}

		zhtest.AssertNotNil(t, cfg.Encodings)
		zhtest.AssertEqual(t, 0, len(cfg.Encodings))
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := Config{
			Encodings:     nil,
			ExcludedPaths: nil,
			IncludedPaths: nil,
		}

		zhtest.AssertNil(t, cfg.Encodings)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("case sensitivity", func(t *testing.T) {
		encodings := []string{"gzip", "GZIP", "Gzip", "br", "BR"}
		cfg := Config{
			Encodings: encodings,
		}
		zhtest.AssertEqual(t, 5, len(cfg.Encodings))
		for i, expectedEncoding := range encodings {
			zhtest.AssertEqual(t, expectedEncoding, cfg.Encodings[i])
		}
	})

	t.Run("duplicate encodings", func(t *testing.T) {
		encodings := []string{"gzip", "deflate", "gzip", "br", "deflate"}
		cfg := Config{
			Encodings: encodings,
		}
		zhtest.AssertEqual(t, 5, len(cfg.Encodings))
		for i, expectedEncoding := range encodings {
			zhtest.AssertEqual(t, expectedEncoding, cfg.Encodings[i])
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		encodings := []string{"", "gzip", ""}
		excludedPaths := []string{"", "/health", ""}
		cfg := Config{
			Encodings:     encodings,
			ExcludedPaths: excludedPaths,
		}

		zhtest.AssertEqual(t, 3, len(cfg.Encodings))
		for i, expectedEncoding := range encodings {
			zhtest.AssertEqual(t, expectedEncoding, cfg.Encodings[i])
		}

		zhtest.AssertEqual(t, 3, len(cfg.ExcludedPaths))
		for i, expectedPath := range excludedPaths {
			zhtest.AssertEqual(t, expectedPath, cfg.ExcludedPaths[i])
		}
	})
}

func TestContentEncodingConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api/v1/upload/*",
			"/static/*",
			"/health",
			"/metrics",
			"*.zip",
			"*.tar.gz",
			"/admin/files/*",
		}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/upload",
			"/static_files",
			"/health-check",
			"/metrics.json",
			"/admin/files (test)",
			"/path with spaces",
			"/path/with/unicode-\xc3\xb1",
		}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})
}

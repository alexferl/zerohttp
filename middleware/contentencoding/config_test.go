package contentencoding

import (
	"reflect"
	"testing"
)

func TestContentEncodingConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	if len(cfg.Encodings) != 2 {
		t.Errorf("expected 2 default encodings, got %d", len(cfg.Encodings))
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default excluded paths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected default included paths to be empty, got %d paths", len(cfg.IncludedPaths))
	}
	expectedEncodings := []string{"gzip", "deflate"}
	if !reflect.DeepEqual(cfg.Encodings, expectedEncodings) {
		t.Errorf("expected default encodings = %v, got %v", expectedEncodings, cfg.Encodings)
	}
	if cfg.Encodings[0] != "gzip" {
		t.Errorf("expected first encoding = 'gzip', got %s", cfg.Encodings[0])
	}
	if cfg.Encodings[1] != "deflate" {
		t.Errorf("expected second encoding = 'deflate', got %s", cfg.Encodings[1])
	}
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
				if len(cfg.Encodings) != len(tt.expected) {
					t.Errorf("expected %d encodings, got %d", len(tt.expected), len(cfg.Encodings))
				}
				if !reflect.DeepEqual(cfg.Encodings, tt.expected) {
					t.Errorf("expected encodings = %v, got %v", tt.expected, cfg.Encodings)
				}
			})
		}
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/upload", "/health", "/metrics", "/static"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 4 {
			t.Errorf("expected 4 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
			t.Errorf("expected included paths = %v, got %v", includedPaths, cfg.IncludedPaths)
		}
	})

	t.Run("single encoding variants", func(t *testing.T) {
		testCases := []string{"gzip", "deflate", "br", "compress", "identity"}
		for _, encoding := range testCases {
			cfg := Config{
				Encodings: []string{encoding},
			}
			if len(cfg.Encodings) != 1 {
				t.Errorf("expected 1 encoding for %s, got %d", encoding, len(cfg.Encodings))
			}
			if cfg.Encodings[0] != encoding {
				t.Errorf("expected encoding = %s, got %s", encoding, cfg.Encodings[0])
			}
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

	if len(cfg.Encodings) != 2 {
		t.Errorf("expected 2 encodings, got %d", len(cfg.Encodings))
	}
	if !reflect.DeepEqual(cfg.Encodings, encodings) {
		t.Error("expected encodings to be set correctly")
	}
	if len(cfg.ExcludedPaths) != 2 {
		t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
	}
	if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
		t.Error("expected excluded paths to be set correctly")
	}
	if len(cfg.IncludedPaths) != 1 {
		t.Errorf("expected 1 allowed path, got %d", len(cfg.IncludedPaths))
	}
	if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
		t.Error("expected included paths to be set correctly")
	}
}

func TestContentEncodingConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := Config{
			Encodings:     []string{},
			ExcludedPaths: []string{},
			IncludedPaths: []string{},
		}

		if cfg.Encodings == nil || len(cfg.Encodings) != 0 {
			t.Errorf("expected empty encodings slice, got %v", cfg.Encodings)
		}
		if cfg.ExcludedPaths == nil || len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected empty excluded paths slice, got %v", cfg.ExcludedPaths)
		}
		if cfg.IncludedPaths == nil || len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected empty included paths slice, got %v", cfg.IncludedPaths)
		}
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := Config{
			Encodings:     nil,
			ExcludedPaths: nil,
			IncludedPaths: nil,
		}

		if cfg.Encodings != nil {
			t.Error("expected encodings to remain nil when nil is passed")
		}
		if cfg.ExcludedPaths != nil {
			t.Error("expected excluded paths to remain nil when nil is passed")
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		encodings := []string{"gzip", "GZIP", "Gzip", "br", "BR"}
		cfg := Config{
			Encodings: encodings,
		}
		if len(cfg.Encodings) != 5 {
			t.Errorf("expected 5 encodings, got %d", len(cfg.Encodings))
		}
		for i, expectedEncoding := range encodings {
			if cfg.Encodings[i] != expectedEncoding {
				t.Errorf("expected encoding[%d] = %s, got %s", i, expectedEncoding, cfg.Encodings[i])
			}
		}
	})

	t.Run("duplicate encodings", func(t *testing.T) {
		encodings := []string{"gzip", "deflate", "gzip", "br", "deflate"}
		cfg := Config{
			Encodings: encodings,
		}
		if len(cfg.Encodings) != 5 {
			t.Errorf("expected 5 encodings (including duplicates), got %d", len(cfg.Encodings))
		}
		for i, expectedEncoding := range encodings {
			if cfg.Encodings[i] != expectedEncoding {
				t.Errorf("expected encoding[%d] = %s, got %s", i, expectedEncoding, cfg.Encodings[i])
			}
		}
	})

	t.Run("empty string values", func(t *testing.T) {
		encodings := []string{"", "gzip", ""}
		excludedPaths := []string{"", "/health", ""}
		cfg := Config{
			Encodings:     encodings,
			ExcludedPaths: excludedPaths,
		}

		if len(cfg.Encodings) != 3 {
			t.Errorf("expected 3 encodings, got %d", len(cfg.Encodings))
		}
		for i, expectedEncoding := range encodings {
			if cfg.Encodings[i] != expectedEncoding {
				t.Errorf("expected encoding[%d] = %q, got %q", i, expectedEncoding, cfg.Encodings[i])
			}
		}

		if len(cfg.ExcludedPaths) != 3 {
			t.Errorf("expected 3 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		for i, expectedPath := range excludedPaths {
			if cfg.ExcludedPaths[i] != expectedPath {
				t.Errorf("expected excluded path[%d] = %q, got %q", i, expectedPath, cfg.ExcludedPaths[i])
			}
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
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/upload",
			"/static_files",
			"/health-check",
			"/metrics.json",
			"/admin/files (test)",
			"/path with spaces",
			"/path/with/unicode-ñ",
		}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})
}

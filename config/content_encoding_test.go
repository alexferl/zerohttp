package config

import (
	"reflect"
	"testing"
)

func TestContentEncodingConfig_DefaultValues(t *testing.T) {
	cfg := DefaultContentEncodingConfig
	if len(cfg.Encodings) != 2 {
		t.Errorf("expected 2 default encodings, got %d", len(cfg.Encodings))
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
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

func TestContentEncodingOptions(t *testing.T) {
	t.Run("encodings option", func(t *testing.T) {
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
				cfg := DefaultContentEncodingConfig
				WithContentEncodingEncodings(tt.input)(&cfg)
				if len(cfg.Encodings) != len(tt.expected) {
					t.Errorf("expected %d encodings, got %d", len(tt.expected), len(cfg.Encodings))
				}
				if !reflect.DeepEqual(cfg.Encodings, tt.expected) {
					t.Errorf("expected encodings = %v, got %v", tt.expected, cfg.Encodings)
				}
			})
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/api/upload", "/health", "/metrics", "/static"}
		cfg := DefaultContentEncodingConfig
		WithContentEncodingExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("single encoding variants", func(t *testing.T) {
		testCases := []string{"gzip", "deflate", "br", "compress", "identity"}
		for _, encoding := range testCases {
			cfg := DefaultContentEncodingConfig
			WithContentEncodingEncodings([]string{encoding})(&cfg)
			if len(cfg.Encodings) != 1 {
				t.Errorf("expected 1 encoding for %s, got %d", encoding, len(cfg.Encodings))
			}
			if cfg.Encodings[0] != encoding {
				t.Errorf("expected encoding = %s, got %s", encoding, cfg.Encodings[0])
			}
		}
	})
}

func TestContentEncodingConfig_MultipleOptions(t *testing.T) {
	encodings := []string{"br", "gzip"}
	exemptPaths := []string{"/upload", "/download"}
	cfg := DefaultContentEncodingConfig
	WithContentEncodingEncodings(encodings)(&cfg)
	WithContentEncodingExemptPaths(exemptPaths)(&cfg)

	if len(cfg.Encodings) != 2 {
		t.Errorf("expected 2 encodings, got %d", len(cfg.Encodings))
	}
	if !reflect.DeepEqual(cfg.Encodings, encodings) {
		t.Error("expected encodings to be set correctly")
	}
	if len(cfg.ExemptPaths) != 2 {
		t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
	}
	if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
		t.Error("expected exempt paths to be set correctly")
	}
}

func TestContentEncodingConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := DefaultContentEncodingConfig
		WithContentEncodingEncodings([]string{})(&cfg)
		WithContentEncodingExemptPaths([]string{})(&cfg)

		if cfg.Encodings == nil || len(cfg.Encodings) != 0 {
			t.Errorf("expected empty encodings slice, got %v", cfg.Encodings)
		}
		if cfg.ExemptPaths == nil || len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %v", cfg.ExemptPaths)
		}
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := DefaultContentEncodingConfig
		WithContentEncodingEncodings(nil)(&cfg)
		WithContentEncodingExemptPaths(nil)(&cfg)

		if cfg.Encodings != nil {
			t.Error("expected encodings to remain nil when nil is passed")
		}
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		encodings := []string{"gzip", "GZIP", "Gzip", "br", "BR"}
		cfg := DefaultContentEncodingConfig
		WithContentEncodingEncodings(encodings)(&cfg)
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
		cfg := DefaultContentEncodingConfig
		WithContentEncodingEncodings(encodings)(&cfg)
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
		exemptPaths := []string{"", "/health", ""}
		cfg := DefaultContentEncodingConfig
		WithContentEncodingEncodings(encodings)(&cfg)
		WithContentEncodingExemptPaths(exemptPaths)(&cfg)

		if len(cfg.Encodings) != 3 {
			t.Errorf("expected 3 encodings, got %d", len(cfg.Encodings))
		}
		for i, expectedEncoding := range encodings {
			if cfg.Encodings[i] != expectedEncoding {
				t.Errorf("expected encoding[%d] = %q, got %q", i, expectedEncoding, cfg.Encodings[i])
			}
		}

		if len(cfg.ExemptPaths) != 3 {
			t.Errorf("expected 3 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		for i, expectedPath := range exemptPaths {
			if cfg.ExemptPaths[i] != expectedPath {
				t.Errorf("expected exempt path[%d] = %q, got %q", i, expectedPath, cfg.ExemptPaths[i])
			}
		}
	})
}

func TestContentEncodingConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api/v1/upload/*",
			"/static/*",
			"/health",
			"/metrics",
			"*.zip",
			"*.tar.gz",
			"/admin/files/*",
		}
		cfg := DefaultContentEncodingConfig
		WithContentEncodingExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api-v1/upload",
			"/static_files",
			"/health-check",
			"/metrics.json",
			"/admin/files (test)",
			"/path with spaces",
			"/path/with/unicode-ñ",
		}
		cfg := DefaultContentEncodingConfig
		WithContentEncodingExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

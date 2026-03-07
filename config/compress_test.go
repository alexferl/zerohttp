package config

import (
	"reflect"
	"testing"
)

func TestCompressConfig_DefaultValues(t *testing.T) {
	cfg := DefaultCompressConfig
	if cfg.Level != 6 {
		t.Errorf("expected default level = 6, got %d", cfg.Level)
	}
	if len(cfg.Types) != 11 {
		t.Errorf("expected 11 default MIME types, got %d", len(cfg.Types))
	}
	if len(cfg.Algorithms) != 2 {
		t.Errorf("expected 2 default algorithms, got %d", len(cfg.Algorithms))
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}

	expectedAlgorithms := []CompressionAlgorithm{Gzip, Deflate}
	if !reflect.DeepEqual(cfg.Algorithms, expectedAlgorithms) {
		t.Errorf("expected default algorithms = %v, got %v", expectedAlgorithms, cfg.Algorithms)
	}

	commonTypes := []string{"text/html", "text/css", "application/javascript", "application/json", "text/plain"}
	for _, expectedType := range commonTypes {
		found := false
		for _, actualType := range cfg.Types {
			if actualType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected common MIME type %s to be included in defaults", expectedType)
		}
	}
}

func TestCompressOptions(t *testing.T) {
	t.Run("level option", func(t *testing.T) {
		cfg := DefaultCompressConfig
		WithCompressLevel(9)(&cfg)
		if cfg.Level != 9 {
			t.Errorf("expected level = 9, got %d", cfg.Level)
		}
	})

	t.Run("types option", func(t *testing.T) {
		types := []string{"text/html", "application/json", "text/css", "application/xml"}
		cfg := DefaultCompressConfig
		WithCompressTypes(types)(&cfg)
		if !reflect.DeepEqual(cfg.Types, types) {
			t.Errorf("expected types = %v, got %v", types, cfg.Types)
		}
	})

	t.Run("algorithms option", func(t *testing.T) {
		algorithms := []CompressionAlgorithm{Gzip}
		cfg := DefaultCompressConfig
		WithCompressAlgorithms(algorithms)(&cfg)
		if !reflect.DeepEqual(cfg.Algorithms, algorithms) {
			t.Errorf("expected algorithms = %v, got %v", algorithms, cfg.Algorithms)
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/api/stream", "/download", "/static/images", "/videos"}
		cfg := DefaultCompressConfig
		WithCompressExemptPaths(exemptPaths)(&cfg)
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestCompressConfig_MultipleOptions(t *testing.T) {
	types := []string{"application/json", "text/html", "text/css"}
	algorithms := []CompressionAlgorithm{Deflate}
	exemptPaths := []string{"/large-files", "/api/binary"}

	cfg := DefaultCompressConfig
	WithCompressLevel(3)(&cfg)
	WithCompressTypes(types)(&cfg)
	WithCompressAlgorithms(algorithms)(&cfg)
	WithCompressExemptPaths(exemptPaths)(&cfg)

	if cfg.Level != 3 {
		t.Errorf("expected level = 3, got %d", cfg.Level)
	}
	if !reflect.DeepEqual(cfg.Types, types) {
		t.Error("expected MIME types to be set correctly")
	}
	if len(cfg.Algorithms) != 1 || cfg.Algorithms[0] != Deflate {
		t.Errorf("expected algorithm = %s, got %v", Deflate, cfg.Algorithms)
	}
	if len(cfg.ExemptPaths) != 2 {
		t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
	}
}

func TestCompressConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := DefaultCompressConfig
		WithCompressTypes([]string{})(&cfg)
		WithCompressAlgorithms([]CompressionAlgorithm{})(&cfg)
		WithCompressExemptPaths([]string{})(&cfg)

		if cfg.Types == nil || len(cfg.Types) != 0 {
			t.Errorf("expected empty types slice, got %v", cfg.Types)
		}
		if cfg.Algorithms == nil || len(cfg.Algorithms) != 0 {
			t.Errorf("expected empty algorithms slice, got %v", cfg.Algorithms)
		}
		if cfg.ExemptPaths == nil || len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %v", cfg.ExemptPaths)
		}
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := DefaultCompressConfig
		WithCompressTypes(nil)(&cfg)
		WithCompressAlgorithms(nil)(&cfg)
		WithCompressExemptPaths(nil)(&cfg)

		if cfg.Types != nil {
			t.Error("expected types to remain nil when nil is passed")
		}
		if cfg.Algorithms != nil {
			t.Error("expected algorithms to remain nil when nil is passed")
		}
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("boundary levels", func(t *testing.T) {
		testCases := []int{-1, 0, 1, 5, 9, 10}
		for _, level := range testCases {
			cfg := DefaultCompressConfig
			WithCompressLevel(level)(&cfg)
			if cfg.Level != level {
				t.Errorf("WithCompressLevel(%d): expected level = %d, got %d", level, level, cfg.Level)
			}
		}
	})
}

func TestCompressConfig_CustomScenarios(t *testing.T) {
	t.Run("mixed algorithms", func(t *testing.T) {
		algorithms := []CompressionAlgorithm{Deflate, Gzip}
		cfg := DefaultCompressConfig
		WithCompressAlgorithms(algorithms)(&cfg)
		if len(cfg.Algorithms) != 2 {
			t.Errorf("expected 2 algorithms, got %d", len(cfg.Algorithms))
		}
		if cfg.Algorithms[0] != Deflate || cfg.Algorithms[1] != Gzip {
			t.Errorf("expected algorithms [Deflate, Gzip], got %v", cfg.Algorithms)
		}
	})

	t.Run("custom MIME types", func(t *testing.T) {
		customTypes := []string{
			"application/vnd.api+json",
			"text/markdown",
			"application/ld+json",
			"text/csv",
		}
		cfg := DefaultCompressConfig
		WithCompressTypes(customTypes)(&cfg)
		if !reflect.DeepEqual(cfg.Types, customTypes) {
			t.Errorf("expected custom types %v, got %v", customTypes, cfg.Types)
		}
	})

	t.Run("path patterns", func(t *testing.T) {
		exemptPaths := []string{
			"/api/v1/upload/*",
			"/static/images/*",
			"/download/*",
			"*.zip",
			"*.gz",
			"/health",
		}
		cfg := DefaultCompressConfig
		WithCompressExemptPaths(exemptPaths)(&cfg)
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

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

func TestCompressConfig_StructAssignment(t *testing.T) {
	t.Run("level assignment", func(t *testing.T) {
		cfg := CompressConfig{
			Level:       9,
			Types:       DefaultCompressConfig.Types,
			Algorithms:  DefaultCompressConfig.Algorithms,
			ExemptPaths: []string{},
		}
		if cfg.Level != 9 {
			t.Errorf("expected level = 9, got %d", cfg.Level)
		}
	})

	t.Run("types assignment", func(t *testing.T) {
		types := []string{"text/html", "application/json", "text/css", "application/xml"}
		cfg := CompressConfig{
			Level:       6,
			Types:       types,
			Algorithms:  DefaultCompressConfig.Algorithms,
			ExemptPaths: []string{},
		}
		if !reflect.DeepEqual(cfg.Types, types) {
			t.Errorf("expected types = %v, got %v", types, cfg.Types)
		}
	})

	t.Run("algorithms assignment", func(t *testing.T) {
		algorithms := []CompressionAlgorithm{Gzip}
		cfg := CompressConfig{
			Level:       6,
			Types:       DefaultCompressConfig.Types,
			Algorithms:  algorithms,
			ExemptPaths: []string{},
		}
		if !reflect.DeepEqual(cfg.Algorithms, algorithms) {
			t.Errorf("expected algorithms = %v, got %v", algorithms, cfg.Algorithms)
		}
	})

	t.Run("exempt paths assignment", func(t *testing.T) {
		exemptPaths := []string{"/api/stream", "/download", "/static/images", "/videos"}
		cfg := CompressConfig{
			Level:       6,
			Types:       DefaultCompressConfig.Types,
			Algorithms:  DefaultCompressConfig.Algorithms,
			ExemptPaths: exemptPaths,
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestCompressConfig_MultipleFields(t *testing.T) {
	types := []string{"application/json", "text/html", "text/css"}
	algorithms := []CompressionAlgorithm{Deflate}
	exemptPaths := []string{"/large-files", "/api/binary"}

	cfg := CompressConfig{
		Level:       3,
		Types:       types,
		Algorithms:  algorithms,
		ExemptPaths: exemptPaths,
	}

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
		cfg := CompressConfig{
			Level:       6,
			Types:       []string{},
			Algorithms:  []CompressionAlgorithm{},
			ExemptPaths: []string{},
		}

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
		cfg := CompressConfig{
			Level:       6,
			Types:       nil,
			Algorithms:  nil,
			ExemptPaths: nil,
		}

		if cfg.Types != nil {
			t.Error("expected types to be nil")
		}
		if cfg.Algorithms != nil {
			t.Error("expected algorithms to be nil")
		}
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to be nil")
		}
	})

	t.Run("boundary levels", func(t *testing.T) {
		testCases := []int{-1, 0, 1, 5, 9, 10}
		for _, level := range testCases {
			cfg := CompressConfig{
				Level:       level,
				Types:       DefaultCompressConfig.Types,
				Algorithms:  DefaultCompressConfig.Algorithms,
				ExemptPaths: []string{},
			}
			if cfg.Level != level {
				t.Errorf("expected level = %d, got %d", level, cfg.Level)
			}
		}
	})
}

func TestCompressConfig_CustomScenarios(t *testing.T) {
	t.Run("mixed algorithms", func(t *testing.T) {
		algorithms := []CompressionAlgorithm{Deflate, Gzip}
		cfg := CompressConfig{
			Level:       6,
			Types:       DefaultCompressConfig.Types,
			Algorithms:  algorithms,
			ExemptPaths: []string{},
		}
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
		cfg := CompressConfig{
			Level:       6,
			Types:       customTypes,
			Algorithms:  DefaultCompressConfig.Algorithms,
			ExemptPaths: []string{},
		}
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
		cfg := CompressConfig{
			Level:       6,
			Types:       DefaultCompressConfig.Types,
			Algorithms:  DefaultCompressConfig.Algorithms,
			ExemptPaths: exemptPaths,
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

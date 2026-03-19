package config

import (
	"reflect"
	"testing"
)

func TestRequestBodySizeConfig_DefaultValues(t *testing.T) {
	cfg := DefaultRequestBodySizeConfig
	if cfg.MaxBytes != 1<<20 {
		t.Errorf("expected default max bytes = %d (1MB), got %d", 1<<20, cfg.MaxBytes)
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default excluded paths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected default included paths to be empty, got %d paths", len(cfg.IncludedPaths))
	}

	// Verify the 1MB calculation
	expectedSize := int64(1048576)
	if cfg.MaxBytes != expectedSize {
		t.Errorf("expected default max bytes = %d bytes, got %d", expectedSize, cfg.MaxBytes)
	}
}

func TestRequestBodySizeConfig_BoundaryValues(t *testing.T) {
	tests := []struct {
		name     string
		maxBytes int64
	}{
		{"zero", 0},
		{"one byte", 1},
		{"1KB", 1024},
		{"1MB", 1048576},
		{"10MB", 10485760},
		{"100MB", 104857600},
		{"1GB", 1073741824},
		{"negative", -1},
		{"max int64", 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := RequestBodySizeConfig{
				MaxBytes:      tt.maxBytes,
				ExcludedPaths: []string{},
			}
			if cfg.MaxBytes != tt.maxBytes {
				t.Errorf("expected max bytes = %d, got %d", tt.maxBytes, cfg.MaxBytes)
			}
		})
	}
}

func TestRequestBodySizeConfig_CommonSizes(t *testing.T) {
	tests := []struct {
		name     string
		maxBytes int64
	}{
		{"1KB", 1 << 10},
		{"4KB", 4 << 10},
		{"64KB", 64 << 10},
		{"1MB", 1 << 20},
		{"2MB", 2 << 20},
		{"5MB", 5 << 20},
		{"10MB", 10 << 20},
		{"100MB", 100 << 20},
		{"1GB", 1 << 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := RequestBodySizeConfig{
				MaxBytes:      tt.maxBytes,
				ExcludedPaths: []string{},
			}
			if cfg.MaxBytes != tt.maxBytes {
				t.Errorf("expected %s max bytes = %d, got %d", tt.name, tt.maxBytes, cfg.MaxBytes)
			}
		})
	}
}

func TestRequestBodySizeConfig_EdgeCases(t *testing.T) {
	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:      1048576,
			ExcludedPaths: []string{},
		}
		if cfg.ExcludedPaths == nil {
			t.Error("expected excluded paths slice to be initialized, not nil")
		}
		if len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected empty excluded paths slice, got %d entries", len(cfg.ExcludedPaths))
		}
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:      1048576,
			ExcludedPaths: nil,
		}
		if cfg.ExcludedPaths != nil {
			t.Error("expected excluded paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		excludedPaths := []string{"", "/upload", ""}
		cfg := RequestBodySizeConfig{
			MaxBytes:      1048576,
			ExcludedPaths: excludedPaths,
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

	t.Run("zero values", func(t *testing.T) {
		cfg := RequestBodySizeConfig{} // Zero values
		if cfg.MaxBytes != 0 {
			t.Errorf("expected zero max bytes = 0, got %d", cfg.MaxBytes)
		}
		if cfg.ExcludedPaths != nil {
			t.Errorf("expected zero excluded paths = nil, got %v", cfg.ExcludedPaths)
		}
		if cfg.IncludedPaths != nil {
			t.Errorf("expected zero included paths = nil, got %v", cfg.IncludedPaths)
		}
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:      1048576,
			IncludedPaths: []string{},
		}
		if cfg.IncludedPaths == nil {
			t.Error("expected included paths slice to be initialized, not nil")
		}
		if len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected empty included paths slice, got %d entries", len(cfg.IncludedPaths))
		}
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:      1048576,
			IncludedPaths: nil,
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
		}
	})

	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := RequestBodySizeConfig{
			MaxBytes:      1048576,
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if !reflect.DeepEqual(cfg.IncludedPaths, includedPaths) {
			t.Errorf("expected included paths = %v, got %v", includedPaths, cfg.IncludedPaths)
		}
	})
}

func TestRequestBodySizeConfig_PathPatterns(t *testing.T) {
	t.Run("path patterns", func(t *testing.T) {
		excludedPaths := []string{
			"/api/v1/upload/*",
			"/files/*",
			"/media/upload",
			"/bulk-import",
			"*.zip",
			"/admin/data/*",
			"/webhooks/large-payload",
		}
		cfg := RequestBodySizeConfig{
			MaxBytes:      10485760, // 10MB
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
			"/files_large",
			"/upload-service",
			"/media.upload",
			"/bulk (import)",
			"/path with spaces",
			"/path/with/unicode-ñ",
			"/files/test@example.com",
		}
		cfg := RequestBodySizeConfig{
			MaxBytes:      5242880, // 5MB
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

func TestRequestBodySizeConfig_StructAssignment(t *testing.T) {
	t.Run("direct struct assignment", func(t *testing.T) {
		excludedPaths := []string{"/upload", "/download"}
		cfg := RequestBodySizeConfig{
			MaxBytes:      5242880, // 5MB
			ExcludedPaths: excludedPaths,
		}

		if cfg.MaxBytes != 5242880 {
			t.Errorf("expected max bytes = 5242880, got %d", cfg.MaxBytes)
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Error("expected excluded paths to be set correctly")
		}
		if len(cfg.ExcludedPaths) != 2 {
			t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
	})

	t.Run("modify struct fields", func(t *testing.T) {
		cfg := DefaultRequestBodySizeConfig

		// Modify fields directly
		cfg.MaxBytes = 2097152 // 2MB
		cfg.ExcludedPaths = []string{"/api/upload", "/files"}

		if cfg.MaxBytes != 2097152 {
			t.Errorf("expected modified max bytes = 2097152, got %d", cfg.MaxBytes)
		}
		if len(cfg.ExcludedPaths) != 2 {
			t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
	})
}

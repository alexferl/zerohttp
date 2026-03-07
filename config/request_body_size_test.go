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
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
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
				MaxBytes:    tt.maxBytes,
				ExemptPaths: []string{},
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
				MaxBytes:    tt.maxBytes,
				ExemptPaths: []string{},
			}
			if cfg.MaxBytes != tt.maxBytes {
				t.Errorf("expected %s max bytes = %d, got %d", tt.name, tt.maxBytes, cfg.MaxBytes)
			}
		})
	}
}

func TestRequestBodySizeConfig_EdgeCases(t *testing.T) {
	t.Run("empty exempt paths", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:    1048576,
			ExemptPaths: []string{},
		}
		if cfg.ExemptPaths == nil {
			t.Error("expected exempt paths slice to be initialized, not nil")
		}
		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %d entries", len(cfg.ExemptPaths))
		}
	})

	t.Run("nil exempt paths", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:    1048576,
			ExemptPaths: nil,
		}
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		exemptPaths := []string{"", "/upload", ""}
		cfg := RequestBodySizeConfig{
			MaxBytes:    1048576,
			ExemptPaths: exemptPaths,
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

	t.Run("zero values", func(t *testing.T) {
		cfg := RequestBodySizeConfig{} // Zero values
		if cfg.MaxBytes != 0 {
			t.Errorf("expected zero max bytes = 0, got %d", cfg.MaxBytes)
		}
		if cfg.ExemptPaths != nil {
			t.Errorf("expected zero exempt paths = nil, got %v", cfg.ExemptPaths)
		}
	})
}

func TestRequestBodySizeConfig_PathPatterns(t *testing.T) {
	t.Run("path patterns", func(t *testing.T) {
		exemptPaths := []string{
			"/api/v1/upload/*",
			"/files/*",
			"/media/upload",
			"/bulk-import",
			"*.zip",
			"/admin/data/*",
			"/webhooks/large-payload",
		}
		cfg := RequestBodySizeConfig{
			MaxBytes:    10485760, // 10MB
			ExemptPaths: exemptPaths,
		}
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
			"/files_large",
			"/upload-service",
			"/media.upload",
			"/bulk (import)",
			"/path with spaces",
			"/path/with/unicode-ñ",
			"/files/test@example.com",
		}
		cfg := RequestBodySizeConfig{
			MaxBytes:    5242880, // 5MB
			ExemptPaths: exemptPaths,
		}
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestRequestBodySizeConfig_StructAssignment(t *testing.T) {
	t.Run("direct struct assignment", func(t *testing.T) {
		exemptPaths := []string{"/upload", "/download"}
		cfg := RequestBodySizeConfig{
			MaxBytes:    5242880, // 5MB
			ExemptPaths: exemptPaths,
		}

		if cfg.MaxBytes != 5242880 {
			t.Errorf("expected max bytes = 5242880, got %d", cfg.MaxBytes)
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Error("expected exempt paths to be set correctly")
		}
		if len(cfg.ExemptPaths) != 2 {
			t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
		}
	})

	t.Run("modify struct fields", func(t *testing.T) {
		cfg := DefaultRequestBodySizeConfig

		// Modify fields directly
		cfg.MaxBytes = 2097152 // 2MB
		cfg.ExemptPaths = []string{"/api/upload", "/files"}

		if cfg.MaxBytes != 2097152 {
			t.Errorf("expected modified max bytes = 2097152, got %d", cfg.MaxBytes)
		}
		if len(cfg.ExemptPaths) != 2 {
			t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
		}
	})
}

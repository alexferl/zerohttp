package requestbodysize

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequestBodySizeConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, int64(1<<20), cfg.MaxBytes)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

	// Verify the 1MB calculation
	expectedSize := int64(1048576)
	zhtest.AssertEqual(t, expectedSize, cfg.MaxBytes)
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
			cfg := Config{
				MaxBytes:      tt.maxBytes,
				ExcludedPaths: []string{},
			}
			zhtest.AssertEqual(t, tt.maxBytes, cfg.MaxBytes)
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
			cfg := Config{
				MaxBytes:      tt.maxBytes,
				ExcludedPaths: []string{},
			}
			zhtest.AssertEqual(t, tt.maxBytes, cfg.MaxBytes)
		})
	}
}

func TestRequestBodySizeConfig_EdgeCases(t *testing.T) {
	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := Config{
			MaxBytes:      1048576,
			ExcludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := Config{
			MaxBytes:      1048576,
			ExcludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.ExcludedPaths)
	})

	t.Run("empty string paths", func(t *testing.T) {
		excludedPaths := []string{"", "/upload", ""}
		cfg := Config{
			MaxBytes:      1048576,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 3, len(cfg.ExcludedPaths))
		for i, expectedPath := range excludedPaths {
			zhtest.AssertEqual(t, expectedPath, cfg.ExcludedPaths[i])
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertEqual(t, int64(0), cfg.MaxBytes)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			MaxBytes:      1048576,
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			MaxBytes:      1048576,
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			MaxBytes:      1048576,
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, includedPaths, cfg.IncludedPaths)
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
		cfg := Config{
			MaxBytes:      10485760, // 10MB
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
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
		cfg := Config{
			MaxBytes:      5242880, // 5MB
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})
}

func TestRequestBodySizeConfig_StructAssignment(t *testing.T) {
	t.Run("direct struct assignment", func(t *testing.T) {
		excludedPaths := []string{"/upload", "/download"}
		cfg := Config{
			MaxBytes:      5242880, // 5MB
			ExcludedPaths: excludedPaths,
		}

		zhtest.AssertEqual(t, int64(5242880), cfg.MaxBytes)
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	})

	t.Run("modify struct fields", func(t *testing.T) {
		cfg := DefaultConfig

		// Modify fields directly
		cfg.MaxBytes = 2097152 // 2MB
		cfg.ExcludedPaths = []string{"/api/upload", "/files"}

		zhtest.AssertEqual(t, int64(2097152), cfg.MaxBytes)
		zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	})
}

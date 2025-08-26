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

func TestRequestBodySizeOptions(t *testing.T) {
	t.Run("max bytes option", func(t *testing.T) {
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeMaxBytes(2097152)(&cfg) // 2MB
		if cfg.MaxBytes != 2097152 {
			t.Errorf("expected max bytes = 2097152, got %d", cfg.MaxBytes)
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/api/upload", "/files", "/media", "/bulk"}
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		exemptPaths := []string{"/upload", "/download"}
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeMaxBytes(5242880)(&cfg) // 5MB
		WithRequestBodySizeExemptPaths(exemptPaths)(&cfg)

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
			cfg := DefaultRequestBodySizeConfig
			WithRequestBodySizeMaxBytes(tt.maxBytes)(&cfg)
			if cfg.MaxBytes != tt.maxBytes {
				t.Errorf("WithRequestBodySizeMaxBytes(%d): expected max bytes = %d, got %d", tt.maxBytes, tt.maxBytes, cfg.MaxBytes)
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
			cfg := DefaultRequestBodySizeConfig
			WithRequestBodySizeMaxBytes(tt.maxBytes)(&cfg)
			if cfg.MaxBytes != tt.maxBytes {
				t.Errorf("expected %s max bytes = %d, got %d", tt.name, tt.maxBytes, cfg.MaxBytes)
			}
		})
	}
}

func TestRequestBodySizeConfig_EdgeCases(t *testing.T) {
	t.Run("empty exempt paths", func(t *testing.T) {
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeExemptPaths([]string{})(&cfg)
		if cfg.ExemptPaths == nil {
			t.Error("expected exempt paths slice to be initialized, not nil")
		}
		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %d entries", len(cfg.ExemptPaths))
		}
	})

	t.Run("nil exempt paths", func(t *testing.T) {
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeExemptPaths(nil)(&cfg)
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		exemptPaths := []string{"", "/upload", ""}
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeExemptPaths(exemptPaths)(&cfg)
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
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeExemptPaths(exemptPaths)(&cfg)
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
		cfg := DefaultRequestBodySizeConfig
		WithRequestBodySizeExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestRequestBodySizeConfigToOptions(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		cfg := RequestBodySizeConfig{
			MaxBytes:    10485760, // 10MB
			ExemptPaths: []string{"/upload", "/files"},
		}
		options := requestBodySizeConfigToOptions(cfg)
		if len(options) != 2 {
			t.Errorf("expected 2 options, got %d", len(options))
		}

		// Apply the options to a new config to test they work correctly
		newCfg := DefaultRequestBodySizeConfig
		for _, option := range options {
			option(&newCfg)
		}
		if newCfg.MaxBytes != 10485760 {
			t.Errorf("expected converted max bytes = 10485760, got %d", newCfg.MaxBytes)
		}
		if !reflect.DeepEqual(newCfg.ExemptPaths, []string{"/upload", "/files"}) {
			t.Errorf("expected converted exempt paths = [/upload /files], got %v", newCfg.ExemptPaths)
		}
	})

	t.Run("default values conversion", func(t *testing.T) {
		cfg := DefaultRequestBodySizeConfig
		options := requestBodySizeConfigToOptions(cfg)
		if len(options) != 2 {
			t.Errorf("expected 2 options for default config, got %d", len(options))
		}

		// Apply options to a fresh config
		newCfg := RequestBodySizeConfig{} // Start with zero values
		for _, option := range options {
			option(&newCfg)
		}
		if newCfg.MaxBytes != DefaultRequestBodySizeConfig.MaxBytes {
			t.Errorf("expected converted max bytes = %d, got %d", DefaultRequestBodySizeConfig.MaxBytes, newCfg.MaxBytes)
		}
		if !reflect.DeepEqual(newCfg.ExemptPaths, DefaultRequestBodySizeConfig.ExemptPaths) {
			t.Errorf("expected converted exempt paths = %v, got %v", DefaultRequestBodySizeConfig.ExemptPaths, newCfg.ExemptPaths)
		}
	})

	t.Run("custom values conversion", func(t *testing.T) {
		tests := []struct {
			name        string
			maxBytes    int64
			exemptPaths []string
		}{
			{"small size few paths", 512000, []string{"/small"}},
			{"large size many paths", 104857600, []string{"/big", "/huge", "/massive"}},
			{"zero size no paths", 0, []string{}},
			{"medium size wildcard paths", 5242880, []string{"/api/*", "*.dat"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := RequestBodySizeConfig{
					MaxBytes:    tt.maxBytes,
					ExemptPaths: tt.exemptPaths,
				}
				options := requestBodySizeConfigToOptions(cfg)

				// Apply to new config
				newCfg := RequestBodySizeConfig{}
				for _, option := range options {
					option(&newCfg)
				}
				if newCfg.MaxBytes != tt.maxBytes {
					t.Errorf("expected max bytes = %d, got %d", tt.maxBytes, newCfg.MaxBytes)
				}
				if !reflect.DeepEqual(newCfg.ExemptPaths, tt.exemptPaths) {
					t.Errorf("expected exempt paths = %v, got %v", tt.exemptPaths, newCfg.ExemptPaths)
				}
			})
		}
	})

	t.Run("options equivalence", func(t *testing.T) {
		originalCfg := RequestBodySizeConfig{
			MaxBytes:    20971520, // 20MB
			ExemptPaths: []string{"/large-upload", "/bulk-data"},
		}

		// Method 1: Apply options individually
		cfg1 := DefaultRequestBodySizeConfig
		WithRequestBodySizeMaxBytes(originalCfg.MaxBytes)(&cfg1)
		WithRequestBodySizeExemptPaths(originalCfg.ExemptPaths)(&cfg1)

		// Method 2: Apply via requestBodySizeConfigToOptions
		cfg2 := DefaultRequestBodySizeConfig
		options := requestBodySizeConfigToOptions(originalCfg)
		for _, option := range options {
			option(&cfg2)
		}

		// Both should be identical
		if !reflect.DeepEqual(cfg1, cfg2) {
			t.Errorf("configurations should be identical: cfg1=%+v, cfg2=%+v", cfg1, cfg2)
		}
	})
}

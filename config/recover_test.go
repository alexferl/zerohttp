package config

import (
	"reflect"
	"testing"
)

func TestRecoverConfig_DefaultValues(t *testing.T) {
	cfg := DefaultRecoverConfig
	if cfg.StackSize != 4<<10 {
		t.Errorf("expected default stack size = %d (4KB), got %d", 4<<10, cfg.StackSize)
	}
	if cfg.EnableStackTrace != true {
		t.Errorf("expected default enable stack trace = true, got %t", cfg.EnableStackTrace)
	}

	// Verify the 4KB calculation
	expectedSize := int64(4096)
	if cfg.StackSize != expectedSize {
		t.Errorf("expected default stack size = %d bytes, got %d", expectedSize, cfg.StackSize)
	}
}

func TestRecoverOptions(t *testing.T) {
	t.Run("stack size option", func(t *testing.T) {
		cfg := DefaultRecoverConfig
		WithRecoverStackSize(8192)(&cfg)
		if cfg.StackSize != 8192 {
			t.Errorf("expected stack size = 8192, got %d", cfg.StackSize)
		}
	})

	t.Run("enable stack trace option", func(t *testing.T) {
		cfg := DefaultRecoverConfig
		WithRecoverEnableStackTrace(false)(&cfg)
		if cfg.EnableStackTrace != false {
			t.Errorf("expected enable stack trace = false, got %t", cfg.EnableStackTrace)
		}
		// Test setting back to true
		WithRecoverEnableStackTrace(true)(&cfg)
		if cfg.EnableStackTrace != true {
			t.Errorf("expected enable stack trace = true, got %t", cfg.EnableStackTrace)
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		cfg := DefaultRecoverConfig
		WithRecoverStackSize(16384)(&cfg)
		WithRecoverEnableStackTrace(false)(&cfg)

		if cfg.StackSize != 16384 {
			t.Errorf("expected stack size = 16384, got %d", cfg.StackSize)
		}
		if cfg.EnableStackTrace != false {
			t.Errorf("expected enable stack trace = false, got %t", cfg.EnableStackTrace)
		}
	})
}

func TestRecoverConfig_StackSizeBoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		stackSize int64
	}{
		{"zero", 0},
		{"one byte", 1},
		{"1KB", 1024},
		{"4KB (default)", 4096},
		{"8KB", 8192},
		{"64KB", 65536},
		{"1MB", 1048576},
		{"negative", -1},
		{"max int64", 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultRecoverConfig
			WithRecoverStackSize(tt.stackSize)(&cfg)
			if cfg.StackSize != tt.stackSize {
				t.Errorf("WithRecoverStackSize(%d): expected stack size = %d, got %d", tt.stackSize, tt.stackSize, cfg.StackSize)
			}
		})
	}
}

func TestRecoverConfig_CommonStackSizes(t *testing.T) {
	tests := []struct {
		name      string
		stackSize int64
	}{
		{"1KB", 1 << 10},
		{"2KB", 2 << 10},
		{"4KB", 4 << 10},
		{"8KB", 8 << 10},
		{"16KB", 16 << 10},
		{"32KB", 32 << 10},
		{"64KB", 64 << 10},
		{"1MB", 1 << 20},
		{"2MB", 2 << 20},
		{"4MB", 4 << 20},
		{"8MB", 8 << 20},
		{"16MB", 16 << 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultRecoverConfig
			WithRecoverStackSize(tt.stackSize)(&cfg)
			if cfg.StackSize != tt.stackSize {
				t.Errorf("expected %s stack size = %d, got %d", tt.name, tt.stackSize, cfg.StackSize)
			}
		})
	}
}

func TestRecoverConfigToOptions(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		cfg := RecoverConfig{
			StackSize:        8192,
			EnableStackTrace: false,
		}
		options := recoverConfigToOptions(cfg)
		if len(options) != 2 {
			t.Errorf("expected 2 options, got %d", len(options))
		}

		// Apply the options to a new config to test they work correctly
		newCfg := DefaultRecoverConfig
		for _, option := range options {
			option(&newCfg)
		}
		if newCfg.StackSize != 8192 {
			t.Errorf("expected converted stack size = 8192, got %d", newCfg.StackSize)
		}
		if newCfg.EnableStackTrace != false {
			t.Errorf("expected converted enable stack trace = false, got %t", newCfg.EnableStackTrace)
		}
	})

	t.Run("default values conversion", func(t *testing.T) {
		cfg := DefaultRecoverConfig
		options := recoverConfigToOptions(cfg)
		if len(options) != 2 {
			t.Errorf("expected 2 options for default config, got %d", len(options))
		}

		// Apply options to a fresh config
		newCfg := RecoverConfig{} // Start with zero values
		for _, option := range options {
			option(&newCfg)
		}
		if newCfg.StackSize != DefaultRecoverConfig.StackSize {
			t.Errorf("expected converted stack size = %d, got %d", DefaultRecoverConfig.StackSize, newCfg.StackSize)
		}
		if newCfg.EnableStackTrace != DefaultRecoverConfig.EnableStackTrace {
			t.Errorf("expected converted enable stack trace = %t, got %t", DefaultRecoverConfig.EnableStackTrace, newCfg.EnableStackTrace)
		}
	})

	t.Run("custom values conversion", func(t *testing.T) {
		tests := []struct {
			name             string
			stackSize        int64
			enableStackTrace bool
		}{
			{"small stack disabled", 1024, false},
			{"large stack enabled", 65536, true},
			{"zero stack disabled", 0, false},
			{"medium stack enabled", 16384, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := RecoverConfig{
					StackSize:        tt.stackSize,
					EnableStackTrace: tt.enableStackTrace,
				}
				options := recoverConfigToOptions(cfg)

				// Apply to new config
				newCfg := RecoverConfig{}
				for _, option := range options {
					option(&newCfg)
				}
				if newCfg.StackSize != tt.stackSize {
					t.Errorf("expected stack size = %d, got %d", tt.stackSize, newCfg.StackSize)
				}
				if newCfg.EnableStackTrace != tt.enableStackTrace {
					t.Errorf("expected enable stack trace = %t, got %t", tt.enableStackTrace, newCfg.EnableStackTrace)
				}
			})
		}
	})

	t.Run("options equivalence", func(t *testing.T) {
		originalCfg := RecoverConfig{
			StackSize:        32768,
			EnableStackTrace: false,
		}

		// Method 1: Apply options individually
		cfg1 := DefaultRecoverConfig
		WithRecoverStackSize(originalCfg.StackSize)(&cfg1)
		WithRecoverEnableStackTrace(originalCfg.EnableStackTrace)(&cfg1)

		// Method 2: Apply via recoverConfigToOptions
		cfg2 := DefaultRecoverConfig
		options := recoverConfigToOptions(originalCfg)
		for _, option := range options {
			option(&cfg2)
		}

		// Both should be identical
		if !reflect.DeepEqual(cfg1, cfg2) {
			t.Errorf("configurations should be identical: cfg1=%+v, cfg2=%+v", cfg1, cfg2)
		}
	})
}

func TestRecoverConfig_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cfg := RecoverConfig{} // Zero values
		if cfg.StackSize != 0 {
			t.Errorf("expected zero stack size = 0, got %d", cfg.StackSize)
		}
		if cfg.EnableStackTrace != false {
			t.Errorf("expected zero enable stack trace = false, got %t", cfg.EnableStackTrace)
		}
	})

	t.Run("boolean toggling", func(t *testing.T) {
		cfg := DefaultRecoverConfig
		// Start with default (true)
		if cfg.EnableStackTrace != true {
			t.Error("expected initial EnableStackTrace = true")
		}
		// Toggle to false
		WithRecoverEnableStackTrace(false)(&cfg)
		if cfg.EnableStackTrace != false {
			t.Error("expected EnableStackTrace = false after toggle")
		}
		// Toggle back to true
		WithRecoverEnableStackTrace(true)(&cfg)
		if cfg.EnableStackTrace != true {
			t.Error("expected EnableStackTrace = true after toggle back")
		}
	})
}

func TestRecoverConfig_UsageScenarios(t *testing.T) {
	tests := []struct {
		name             string
		stackSize        int64
		enableStackTrace bool
		description      string
	}{
		{"debugging enabled", 8192, true, "large stack with tracing for debugging"},
		{"production minimal", 1024, false, "small stack without tracing for production"},
		{"development verbose", 16384, true, "very large stack with tracing for development"},
		{"performance optimized", 0, false, "no stack allocation for performance"},
		{"security focused", 2048, false, "moderate stack without exposing traces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultRecoverConfig
			WithRecoverStackSize(tt.stackSize)(&cfg)
			WithRecoverEnableStackTrace(tt.enableStackTrace)(&cfg)

			if cfg.StackSize != tt.stackSize {
				t.Errorf("%s: expected stack size = %d, got %d", tt.description, tt.stackSize, cfg.StackSize)
			}
			if cfg.EnableStackTrace != tt.enableStackTrace {
				t.Errorf("%s: expected enable stack trace = %t, got %t", tt.description, tt.enableStackTrace, cfg.EnableStackTrace)
			}
		})
	}
}

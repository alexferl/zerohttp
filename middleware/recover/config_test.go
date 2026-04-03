package recover

import (
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRecoverConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, int64(4<<10), cfg.StackSize)
	zhtest.AssertTrue(t, *cfg.EnableStackTrace)

	// Verify the 4KB calculation
	expectedSize := int64(4096)
	zhtest.AssertEqual(t, expectedSize, cfg.StackSize)
	zhtest.AssertEqual(t, int64(4096), cfg.StackSize)
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
			cfg := Config{
				StackSize:        tt.stackSize,
				EnableStackTrace: config.Bool(true),
			}
			zhtest.AssertEqual(t, tt.stackSize, cfg.StackSize)
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
			cfg := Config{
				StackSize:        tt.stackSize,
				EnableStackTrace: config.Bool(true),
			}
			zhtest.AssertEqual(t, tt.stackSize, cfg.StackSize)
		})
	}
}

func TestRecoverConfig_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertEqual(t, int64(0), cfg.StackSize)
		zhtest.AssertNil(t, cfg.EnableStackTrace)
	})

	t.Run("boolean toggling", func(t *testing.T) {
		cfg := Config{
			StackSize:        4096,
			EnableStackTrace: config.Bool(true),
		}
		// Start with true
		zhtest.AssertTrue(t, *cfg.EnableStackTrace)
		// Toggle to false
		cfg.EnableStackTrace = config.Bool(false)
		zhtest.AssertFalse(t, *cfg.EnableStackTrace)
		// Toggle back to true
		cfg.EnableStackTrace = config.Bool(true)
		zhtest.AssertTrue(t, *cfg.EnableStackTrace)
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
			cfg := Config{
				StackSize:        tt.stackSize,
				EnableStackTrace: config.Bool(tt.enableStackTrace),
			}

			zhtest.AssertEqual(t, tt.stackSize, cfg.StackSize)
			zhtest.AssertEqual(t, tt.enableStackTrace, *cfg.EnableStackTrace)
		})
	}
}

func TestRecoverConfig_StructAssignment(t *testing.T) {
	t.Run("direct struct assignment", func(t *testing.T) {
		cfg := Config{
			StackSize:        8192,
			EnableStackTrace: config.Bool(false),
		}

		zhtest.AssertEqual(t, int64(8192), cfg.StackSize)
		zhtest.AssertFalse(t, *cfg.EnableStackTrace)
	})

	t.Run("modify struct fields", func(t *testing.T) {
		cfg := DefaultConfig

		// Modify fields directly
		cfg.StackSize = 16384
		cfg.EnableStackTrace = config.Bool(false)

		zhtest.AssertEqual(t, int64(16384), cfg.StackSize)
		zhtest.AssertFalse(t, *cfg.EnableStackTrace)
	})
}

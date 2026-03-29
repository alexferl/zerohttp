package compress

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestCompressConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, 6, cfg.Level)
	zhtest.AssertEqual(t, 11, len(cfg.Types))
	zhtest.AssertEqual(t, 2, len(cfg.Algorithms))
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

	expectedAlgorithms := []Algorithm{Gzip, Deflate}
	zhtest.AssertEqual(t, expectedAlgorithms, cfg.Algorithms)

	commonTypes := []string{"text/html", "text/css", "application/javascript", "application/json", "text/plain"}
	for _, expectedType := range commonTypes {
		found := false
		for _, actualType := range cfg.Types {
			if actualType == expectedType {
				found = true
				break
			}
		}
		zhtest.AssertTrue(t, found)
	}
}

func TestCompressConfig_StructAssignment(t *testing.T) {
	t.Run("level assignment", func(t *testing.T) {
		cfg := Config{
			Level:         9,
			Types:         DefaultConfig.Types,
			Algorithms:    DefaultConfig.Algorithms,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, 9, cfg.Level)
	})

	t.Run("types assignment", func(t *testing.T) {
		types := []string{"text/html", "application/json", "text/css", "application/xml"}
		cfg := Config{
			Level:         6,
			Types:         types,
			Algorithms:    DefaultConfig.Algorithms,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, types, cfg.Types)
	})

	t.Run("algorithms assignment", func(t *testing.T) {
		algorithms := []Algorithm{Gzip}
		cfg := Config{
			Level:         6,
			Types:         DefaultConfig.Types,
			Algorithms:    algorithms,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, algorithms, cfg.Algorithms)
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/api/stream", "/download", "/static/images", "/videos"}
		cfg := Config{
			Level:         6,
			Types:         DefaultConfig.Types,
			Algorithms:    DefaultConfig.Algorithms,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths assignment", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			Level:         6,
			Types:         DefaultConfig.Types,
			Algorithms:    DefaultConfig.Algorithms,
			ExcludedPaths: []string{},
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, includedPaths, cfg.IncludedPaths)
	})
}

func TestCompressConfig_MultipleFields(t *testing.T) {
	types := []string{"application/json", "text/html", "text/css"}
	algorithms := []Algorithm{Deflate}
	excludedPaths := []string{"/large-files", "/api/binary"}

	cfg := Config{
		Level:         3,
		Types:         types,
		Algorithms:    algorithms,
		ExcludedPaths: excludedPaths,
	}

	zhtest.AssertEqual(t, 3, cfg.Level)
	zhtest.AssertEqual(t, types, cfg.Types)
	zhtest.AssertEqual(t, 1, len(cfg.Algorithms))
	zhtest.AssertEqual(t, Deflate, cfg.Algorithms[0])
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
}

func TestCompressConfig_EdgeCases(t *testing.T) {
	t.Run("empty slices", func(t *testing.T) {
		cfg := Config{
			Level:         6,
			Types:         []string{},
			Algorithms:    []Algorithm{},
			ExcludedPaths: []string{},
			IncludedPaths: []string{},
		}

		zhtest.AssertNotNil(t, cfg.Types)
		zhtest.AssertEqual(t, 0, len(cfg.Types))
		zhtest.AssertNotNil(t, cfg.Algorithms)
		zhtest.AssertEqual(t, 0, len(cfg.Algorithms))
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil slices", func(t *testing.T) {
		cfg := Config{
			Level:         6,
			Types:         nil,
			Algorithms:    nil,
			ExcludedPaths: nil,
			IncludedPaths: nil,
		}

		zhtest.AssertNil(t, cfg.Types)
		zhtest.AssertNil(t, cfg.Algorithms)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("boundary levels", func(t *testing.T) {
		testCases := []int{-1, 0, 1, 5, 9, 10}
		for _, level := range testCases {
			cfg := Config{
				Level:         level,
				Types:         DefaultConfig.Types,
				Algorithms:    DefaultConfig.Algorithms,
				ExcludedPaths: []string{},
			}
			zhtest.AssertEqual(t, level, cfg.Level)
		}
	})
}

func TestCompressConfig_CustomScenarios(t *testing.T) {
	t.Run("mixed algorithms", func(t *testing.T) {
		algorithms := []Algorithm{Deflate, Gzip}
		cfg := Config{
			Level:         6,
			Types:         DefaultConfig.Types,
			Algorithms:    algorithms,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, 2, len(cfg.Algorithms))
		zhtest.AssertEqual(t, Deflate, cfg.Algorithms[0])
		zhtest.AssertEqual(t, Gzip, cfg.Algorithms[1])
	})

	t.Run("custom MIME types", func(t *testing.T) {
		customTypes := []string{
			"application/vnd.api+json",
			"text/markdown",
			"application/ld+json",
			"text/csv",
		}
		cfg := Config{
			Level:         6,
			Types:         customTypes,
			Algorithms:    DefaultConfig.Algorithms,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, customTypes, cfg.Types)
	})

	t.Run("path patterns", func(t *testing.T) {
		excludedPaths := []string{
			"/api/v1/upload/*",
			"/static/images/*",
			"/download/*",
			"*.zip",
			"*.gz",
			"/health",
		}
		cfg := Config{
			Level:         6,
			Types:         DefaultConfig.Types,
			Algorithms:    DefaultConfig.Algorithms,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})
}

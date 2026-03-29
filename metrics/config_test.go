package metrics

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultMetricsConfig(t *testing.T) {
	defaults := DefaultConfig

	zhtest.AssertEqual(t, "/metrics", defaults.Endpoint)
	zhtest.AssertNotNil(t, defaults.ServerAddr)
	zhtest.AssertEqual(t, "localhost:9090", *defaults.ServerAddr)

	expectedDurationBuckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	zhtest.AssertEqual(t, expectedDurationBuckets, defaults.DurationBuckets)

	expectedSizeBuckets := []float64{100, 1000, 10000, 100000, 1000000, 10000000}
	zhtest.AssertEqual(t, expectedSizeBuckets, defaults.SizeBuckets)

	expectedExcludedPaths := []string{"/metrics"}
	zhtest.AssertEqual(t, expectedExcludedPaths, defaults.ExcludedPaths)

	zhtest.AssertNotNil(t, defaults.PathLabelFunc)
	result := defaults.PathLabelFunc("/api/users/123")
	zhtest.AssertEqual(t, "/api/users/123", result)

	zhtest.AssertNil(t, defaults.CustomLabels)
	zhtest.AssertEqual(t, 0, len(defaults.IncludedPaths))
}

func TestMetricsConfig_CustomLabels(t *testing.T) {
	customLabels := func(r *http.Request) map[string]string {
		return map[string]string{
			"region": "us-east-1",
		}
	}

	cfg := Config{
		Endpoint:        "/custom-metrics",
		ServerAddr:      config.String("localhost:9091"),
		DurationBuckets: []float64{0.1, 0.5, 1.0},
		SizeBuckets:     []float64{1000, 10000},
		ExcludedPaths:   []string{"/health", "/readyz"},
		PathLabelFunc:   func(p string) string { return "/normalized" },
		CustomLabels:    customLabels,
	}

	zhtest.AssertEqual(t, "/custom-metrics", cfg.Endpoint)
	zhtest.AssertNotNil(t, cfg.ServerAddr)
	zhtest.AssertEqual(t, "localhost:9091", *cfg.ServerAddr)
	zhtest.AssertEqual(t, 3, len(cfg.DurationBuckets))
	zhtest.AssertEqual(t, 2, len(cfg.SizeBuckets))
	zhtest.AssertEqual(t, []string{"/health", "/readyz"}, cfg.ExcludedPaths)

	result := cfg.PathLabelFunc("/api/users/123")
	zhtest.AssertEqual(t, "/normalized", result)

	zhtest.AssertNotNil(t, cfg.CustomLabels)
	labels := cfg.CustomLabels(nil)
	zhtest.AssertEqual(t, "us-east-1", labels["region"])
}

func TestMetricsConfig_EmptyServerAddr(t *testing.T) {
	// Empty ServerAddr means metrics are served on the main server
	cfg := Config{
		ServerAddr: config.String(""),
		Endpoint:   "/metrics",
	}

	zhtest.AssertNotNil(t, cfg.ServerAddr)
	zhtest.AssertEqual(t, "", *cfg.ServerAddr)
}

func TestMetricsConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		cfg := Config{
			Endpoint:      "/metrics",
			IncludedPaths: []string{"/api/public", "/health"},
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})
}

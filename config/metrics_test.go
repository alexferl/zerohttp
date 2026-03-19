package config

import (
	"net/http"
	"reflect"
	"testing"
)

func TestDefaultMetricsConfig(t *testing.T) {
	defaults := DefaultMetricsConfig

	if defaults.Endpoint != "/metrics" {
		t.Errorf("expected Endpoint to be /metrics, got %s", defaults.Endpoint)
	}

	if defaults.ServerAddr == nil || *defaults.ServerAddr != "localhost:9090" {
		t.Errorf("expected ServerAddr to be localhost:9090, got %v", defaults.ServerAddr)
	}

	expectedDurationBuckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	if !reflect.DeepEqual(defaults.DurationBuckets, expectedDurationBuckets) {
		t.Errorf("expected DurationBuckets %v, got %v", expectedDurationBuckets, defaults.DurationBuckets)
	}

	expectedSizeBuckets := []float64{100, 1000, 10000, 100000, 1000000, 10000000}
	if !reflect.DeepEqual(defaults.SizeBuckets, expectedSizeBuckets) {
		t.Errorf("expected SizeBuckets %v, got %v", expectedSizeBuckets, defaults.SizeBuckets)
	}

	expectedExcludedPaths := []string{"/metrics"}
	if !reflect.DeepEqual(defaults.ExcludedPaths, expectedExcludedPaths) {
		t.Errorf("expected ExcludedPaths %v, got %v", expectedExcludedPaths, defaults.ExcludedPaths)
	}

	if defaults.PathLabelFunc == nil {
		t.Error("expected PathLabelFunc to be set")
	} else {
		// Test that PathLabelFunc returns the path as-is
		result := defaults.PathLabelFunc("/api/users/123")
		if result != "/api/users/123" {
			t.Errorf("expected PathLabelFunc to return path as-is, got %s", result)
		}
	}

	if defaults.CustomLabels != nil {
		t.Error("expected CustomLabels to be nil by default")
	}
	if len(defaults.IncludedPaths) != 0 {
		t.Errorf("expected IncludedPaths to be empty, got %d paths", len(defaults.IncludedPaths))
	}
}

func TestMetricsConfig_CustomLabels(t *testing.T) {
	customLabels := func(r *http.Request) map[string]string {
		return map[string]string{
			"region": "us-east-1",
		}
	}

	cfg := MetricsConfig{
		Endpoint:        "/custom-metrics",
		ServerAddr:      String("localhost:9091"),
		DurationBuckets: []float64{0.1, 0.5, 1.0},
		SizeBuckets:     []float64{1000, 10000},
		ExcludedPaths:   []string{"/health", "/readyz"},
		PathLabelFunc:   func(p string) string { return "/normalized" },
		CustomLabels:    customLabels,
	}

	if cfg.Endpoint != "/custom-metrics" {
		t.Errorf("expected Endpoint to be /custom-metrics, got %s", cfg.Endpoint)
	}

	if cfg.ServerAddr == nil || *cfg.ServerAddr != "localhost:9091" {
		t.Errorf("expected ServerAddr to be localhost:9091, got %v", cfg.ServerAddr)
	}

	if len(cfg.DurationBuckets) != 3 {
		t.Errorf("expected 3 DurationBuckets, got %d", len(cfg.DurationBuckets))
	}

	if len(cfg.SizeBuckets) != 2 {
		t.Errorf("expected 2 SizeBuckets, got %d", len(cfg.SizeBuckets))
	}

	if !reflect.DeepEqual(cfg.ExcludedPaths, []string{"/health", "/readyz"}) {
		t.Errorf("expected ExcludedPaths [health readyz], got %v", cfg.ExcludedPaths)
	}

	result := cfg.PathLabelFunc("/api/users/123")
	if result != "/normalized" {
		t.Errorf("expected PathLabelFunc to return /normalized, got %s", result)
	}

	if cfg.CustomLabels == nil {
		t.Error("expected CustomLabels to be set")
	} else {
		labels := cfg.CustomLabels(nil)
		if labels["region"] != "us-east-1" {
			t.Errorf("expected region label to be us-east-1, got %s", labels["region"])
		}
	}
}

func TestMetricsConfig_EmptyServerAddr(t *testing.T) {
	// Empty ServerAddr means metrics are served on the main server
	cfg := MetricsConfig{
		ServerAddr: String(""),
		Endpoint:   "/metrics",
	}

	if cfg.ServerAddr == nil || *cfg.ServerAddr != "" {
		t.Errorf("expected ServerAddr to be empty, got %v", cfg.ServerAddr)
	}
}

func TestMetricsConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		cfg := MetricsConfig{
			Endpoint:      "/metrics",
			IncludedPaths: []string{"/api/public", "/health"},
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		if cfg.IncludedPaths[0] != "/api/public" {
			t.Errorf("expected first allowed path to be /api/public, got %s", cfg.IncludedPaths[0])
		}
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := MetricsConfig{
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
		cfg := MetricsConfig{
			IncludedPaths: nil,
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
		}
	})
}

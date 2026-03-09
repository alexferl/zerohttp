package config

import "net/http"

// MetricsConfig allows customization of metrics behavior.
type MetricsConfig struct {
	// Enabled determines if metrics middleware is active.
	// Default: true
	Enabled bool

	// Endpoint is the path where metrics are exposed.
	// Default: "/metrics"
	Endpoint string

	// DurationBuckets defines histogram buckets for request duration (seconds).
	// Default: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	DurationBuckets []float64

	// SizeBuckets defines histogram buckets for request/response size (bytes).
	// Default: []float64{100, 1000, 10000, 100000, 1000000, 10000000}
	SizeBuckets []float64

	// ExcludePaths are paths to exclude from metrics (e.g., health checks).
	// Default: ["/health", "/metrics"]
	ExcludePaths []string

	// PathLabelFunc transforms path for labeling (e.g., normalize IDs).
	// Default: identity function (path used as-is)
	PathLabelFunc func(path string) string

	// CustomLabels allows adding user-defined labels to all metrics.
	// Called per-request to extract dynamic labels.
	// Default: nil
	CustomLabels func(r *http.Request) map[string]string
}

// DefaultMetricsConfig contains default values for metrics configuration.
var DefaultMetricsConfig = MetricsConfig{
	Enabled:         true,
	Endpoint:        "/metrics",
	DurationBuckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	SizeBuckets:     []float64{100, 1000, 10000, 100000, 1000000, 10000000},
	ExcludePaths:    []string{"/health", "/metrics"},
	PathLabelFunc:   func(p string) string { return p },
}

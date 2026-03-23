package metrics

import (
	"net/http"

	"github.com/alexferl/zerohttp/config"
)

// Config allows customization of metrics behavior.
type Config struct {
	// Enabled determines if metrics are collected.
	// nil = use default (enabled), true = enabled, false = disabled
	// Default: nil (enabled)
	Enabled *bool

	// Endpoint is the path where metrics are exposed.
	// Default: "/metrics"
	Endpoint string

	// ServerAddr is the address for a dedicated metrics server.
	// Metrics are served on a separate port bound to localhost for security,
	// preventing exposure of internal metrics to the public internet.
	// Set to empty string (via config.String("")) to serve metrics on the main application server (not recommended).
	// Default: "localhost:9090"
	ServerAddr *string

	// DurationBuckets defines histogram buckets for request duration (seconds).
	// Default: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	DurationBuckets []float64

	// SizeBuckets defines histogram buckets for request/response size (bytes).
	// Default: []float64{100, 1000, 10000, 100000, 1000000, 10000000}
	SizeBuckets []float64

	// ExcludedPaths are paths to exclude from metrics (e.g., health checks).
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: ["/metrics"]
	ExcludedPaths []string

	// IncludedPaths contains paths where metrics collection is explicitly applied.
	// If set, metrics will only be collected for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, metrics collection applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// PathLabelFunc transforms path for labeling (e.g., normalize IDs).
	// Default: identity function (path used as-is)
	PathLabelFunc func(path string) string

	// CustomLabels allows adding user-defined labels to all metrics.
	// Called per-request to extract dynamic labels.
	// Default: nil
	CustomLabels func(r *http.Request) map[string]string
}

// DefaultConfig contains default values for metrics configuration.
var DefaultConfig = Config{
	Endpoint:        "/metrics",
	ServerAddr:      config.String("localhost:9090"),
	DurationBuckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	SizeBuckets:     []float64{100, 1000, 10000, 100000, 1000000, 10000000},
	ExcludedPaths:   []string{"/metrics"},
	IncludedPaths:   []string{},
	PathLabelFunc:   func(p string) string { return p },
}

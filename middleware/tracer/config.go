package tracer

import (
	"net/http"
	"slices"

	"github.com/alexferl/zerohttp/trace"
)

// Config holds configuration for the tracing middleware.
type Config struct {
	TracerField trace.Tracer

	// ExcludedPaths is a list of paths that should not be traced.
	// Requests to these paths will not create spans.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where tracing is explicitly applied.
	// If set, tracing will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, tracing applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// SpanNameFormatter is a custom function to generate span names.
	// If nil, the default formatter is used (returns "{method} {path}").
	// Default: nil
	SpanNameFormatter func(r *http.Request) string
}

// DefaultConfig contains the default values for Config.
var DefaultConfig = Config{
	ExcludedPaths:     []string{},
	IncludedPaths:     []string{},
	SpanNameFormatter: nil,
}

// DefaultSpanNameFormatter returns the default span name for a request.
// Format: "{method} {path}"
func DefaultSpanNameFormatter(r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// ConfigWrapper wraps Config to provide helper methods.
type ConfigWrapper struct {
	Config Config
}

// Wrap creates a new ConfigWrapper.
func (c Config) Wrap() *ConfigWrapper {
	return &ConfigWrapper{Config: c}
}

// IsExcluded checks if a path is excluded from tracing.
func (w *ConfigWrapper) IsExcluded(path string) bool {
	return slices.Contains(w.Config.ExcludedPaths, path)
}

// GetSpanName returns the span name for a request.
func (w *ConfigWrapper) GetSpanName(r *http.Request) string {
	if w.Config.SpanNameFormatter != nil {
		return w.Config.SpanNameFormatter(r)
	}
	return DefaultSpanNameFormatter(r)
}

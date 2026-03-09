package config

import (
	"net/http"
	"slices"

	"github.com/alexferl/zerohttp/trace"
)

// TracerConfig holds configuration for the tracing middleware.
type TracerConfig struct {
	// ExemptPaths is a list of paths that should not be traced.
	// Requests to these paths will not create spans.
	// Default: nil (all paths are traced)
	ExemptPaths []string

	// SpanNameFormatter is a custom function to generate span names.
	// If nil, the default formatter is used (returns "{method} {path}").
	// Default: nil
	SpanNameFormatter func(r *http.Request) string
}

// DefaultTracerConfig contains the default values for TracerConfig.
var DefaultTracerConfig = TracerConfig{
	ExemptPaths:       nil,
	SpanNameFormatter: nil,
}

// DefaultSpanNameFormatter returns the default span name for a request.
// Format: "{method} {path}"
func DefaultSpanNameFormatter(r *http.Request) string {
	return r.Method + " " + r.URL.Path
}

// TracerConfigWrapper wraps TracerConfig to provide helper methods.
type TracerConfigWrapper struct {
	Config TracerConfig
}

// Wrap creates a new TracerConfigWrapper.
func (c TracerConfig) Wrap() *TracerConfigWrapper {
	return &TracerConfigWrapper{Config: c}
}

// IsExempt checks if a path is exempt from tracing.
func (w *TracerConfigWrapper) IsExempt(path string) bool {
	return slices.Contains(w.Config.ExemptPaths, path)
}

// GetSpanName returns the span name for a request.
func (w *TracerConfigWrapper) GetSpanName(r *http.Request) string {
	if w.Config.SpanNameFormatter != nil {
		return w.Config.SpanNameFormatter(r)
	}
	return DefaultSpanNameFormatter(r)
}

// TracerField is a type alias for trace.Tracer to avoid import cycles
// when embedding in Config. The actual field in Config is of type trace.Tracer.
type TracerField = trace.Tracer

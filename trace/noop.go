package trace

import "context"

// NoopTracer is a Tracer implementation that does nothing.
// It is used as the default when no tracer is configured.
type NoopTracer struct{}

// NewNoopTracer creates a new no-op tracer.
func NewNoopTracer() Tracer {
	return &NoopTracer{}
}

// Start returns the context unchanged and a no-op span.
func (n *NoopTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &noopSpan{}
}

// Ensure NoopTracer implements Tracer
var _ Tracer = (*NoopTracer)(nil)

type noopSpan struct{}

func (n *noopSpan) End() {}

func (n *noopSpan) SetStatus(code Code, description string) {}

func (n *noopSpan) SetAttributes(attrs ...Attribute) {}

func (n *noopSpan) RecordError(err error, opts ...ErrorOption) {}

// Ensure noopSpan implements Span
var _ Span = (*noopSpan)(nil)

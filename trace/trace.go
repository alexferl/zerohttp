package trace

import "context"

// Tracer is the interface for creating spans in distributed traces.
// Implementations should be safe for concurrent use.
type Tracer interface {
	// Start creates a new span and returns the updated context containing the span.
	// The span becomes the "current" span in the returned context.
	//
	// The parent span is determined from ctx using SpanFromContext.
	// If no parent span exists in ctx, the new span is a root span.
	//
	// The span must be ended by calling End() on the returned Span.
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// Span represents a single operation within a trace.
type Span interface {
	// End completes the span. No further operations should be performed on the span
	// after End is called. End is safe to call multiple times; subsequent calls
	// are no-ops.
	End()

	// SetStatus sets the status of the span. If used, this should be called before End.
	// code indicates whether the span completed successfully (CodeOk) or with an error (CodeError).
	// description provides additional details when code is CodeError.
	SetStatus(code Code, description string)

	// SetAttributes adds attributes to the span.
	// Attributes are key-value pairs that provide additional context about the span.
	// Duplicate keys overwrite existing values.
	SetAttributes(attrs ...Attribute)

	// RecordError records an error as an exception on the span.
	// This indicates that the operation represented by the span encountered an error.
	RecordError(err error, opts ...ErrorOption)
}

// Code represents the status code of a span.
type Code uint32

const (
	// CodeUnset is the default status code, indicating the span status was not set.
	CodeUnset Code = 0

	// CodeOk indicates the span completed successfully.
	CodeOk Code = 1

	// CodeError indicates the span completed with an error.
	CodeError Code = 2
)

// Attribute is a key-value pair that provides metadata about a span.
type Attribute struct {
	Key   string
	Value any
}

// String creates a string attribute.
func String(key, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

// Int creates an int attribute.
func Int(key string, value int) Attribute {
	return Attribute{Key: key, Value: value}
}

// Int64 creates an int64 attribute.
func Int64(key string, value int64) Attribute {
	return Attribute{Key: key, Value: value}
}

// Float64 creates a float64 attribute.
func Float64(key string, value float64) Attribute {
	return Attribute{Key: key, Value: value}
}

// Bool creates a bool attribute.
func Bool(key string, value bool) Attribute {
	return Attribute{Key: key, Value: value}
}

// SpanOption applies a configuration to a span.
type SpanOption interface {
	apply(*spanConfig)
}

type spanConfig struct {
	attributes []Attribute
}

// WithAttributes returns a SpanOption that sets attributes on the span.
func WithAttributes(attrs ...Attribute) SpanOption {
	return withAttributes(attrs)
}

type withAttributes []Attribute

func (w withAttributes) apply(cfg *spanConfig) {
	cfg.attributes = append(cfg.attributes, w...)
}

// ErrorOption applies a configuration when recording an error.
type ErrorOption interface {
	applyError(*errorConfig)
}

type errorConfig struct {
	attributes []Attribute
}

// WithErrorAttributes returns an ErrorOption that sets attributes on the error.
func WithErrorAttributes(attrs ...Attribute) ErrorOption {
	return withErrorAttributes(attrs)
}

type withErrorAttributes []Attribute

func (w withErrorAttributes) applyError(cfg *errorConfig) {
	cfg.attributes = append(cfg.attributes, w...)
}

// contextKey is the type for context keys used by this package.
type contextKey int

const spanKey contextKey = 0

// ContextWithSpan returns a new context containing the provided span.
// This is useful for passing spans through context to child operations.
func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, spanKey, span)
}

// SpanFromContext returns the span contained in the context, if any.
// Returns nil if no span is found in the context.
func SpanFromContext(ctx context.Context) Span {
	if ctx == nil {
		return nil
	}
	if s, ok := ctx.Value(spanKey).(Span); ok {
		return s
	}
	return nil
}

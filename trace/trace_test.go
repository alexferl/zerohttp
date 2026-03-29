package trace

import (
	"context"
	"errors"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestAttributeHelpers(t *testing.T) {
	tests := []struct {
		name    string
		attr    Attribute
		wantKey string
		wantVal any
	}{
		{"String", String("key", "value"), "key", "value"},
		{"Int", Int("count", 42), "count", 42},
		{"Int64", Int64("size", 9223372036854775807), "size", int64(9223372036854775807)},
		{"Float64", Float64("pi", 3.14), "pi", 3.14},
		{"Bool", Bool("active", true), "active", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zhtest.AssertEqual(t, tt.wantKey, tt.attr.Key)
			zhtest.AssertEqual(t, tt.wantVal, tt.attr.Value)
		})
	}
}

func TestNoopTracer(t *testing.T) {
	tracer := NewNoopTracer()

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span",
		WithAttributes(String("key", "value")),
	)

	// No-op should not panic
	span.SetAttributes(String("another", "attr"))
	span.SetStatus(CodeOk, "success")
	span.RecordError(errors.New("test error"))
	span.End()

	// Context should be unchanged
	zhtest.AssertNotNil(t, ctx)
}

func TestContextWithSpan(t *testing.T) {
	tracer := NewNoopTracer()

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test")

	// Store span in context
	ctx = ContextWithSpan(ctx, span)

	// Retrieve span from context
	retrieved := SpanFromContext(ctx)
	zhtest.AssertNotNil(t, retrieved)

	// Nil context should return nil
	zhtest.AssertNil(t, SpanFromContext(context.TODO()))

	// Context without span should return nil
	emptyCtx := context.Background()
	zhtest.AssertNil(t, SpanFromContext(emptyCtx))
}

func TestSpanConfig(t *testing.T) {
	cfg := &spanConfig{}

	// Apply WithAttributes
	opt := WithAttributes(
		String("key1", "val1"),
		Int("key2", 42),
	)
	opt.apply(cfg)

	zhtest.AssertEqual(t, 2, len(cfg.attributes))
}

func TestErrorConfig(t *testing.T) {
	cfg := &errorConfig{}

	// Apply WithErrorAttributes
	opt := WithErrorAttributes(String("error.type", "test"))
	opt.applyError(cfg)

	zhtest.AssertEqual(t, 1, len(cfg.attributes))
}

func TestCodeConstants(t *testing.T) {
	zhtest.AssertEqual(t, 0, int(CodeUnset))
	zhtest.AssertEqual(t, 1, int(CodeOk))
	zhtest.AssertEqual(t, 2, int(CodeError))
}

// mockTracer is a test tracer that records span creation
type mockTracer struct {
	spans []*mockSpan
}

func (m *mockTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	span := &mockSpan{name: name}
	m.spans = append(m.spans, span)

	// Apply options
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	span.attributes = cfg.attributes

	return ContextWithSpan(ctx, span), span
}

type mockSpan struct {
	name       string
	statusCode Code
	statusDesc string
	attributes []Attribute
	errors     []error
	ended      bool
}

func (m *mockSpan) End() {
	m.ended = true
}

func (m *mockSpan) SetStatus(code Code, description string) {
	m.statusCode = code
	m.statusDesc = description
}

func (m *mockSpan) SetAttributes(attrs ...Attribute) {
	m.attributes = append(m.attributes, attrs...)
}

func (m *mockSpan) RecordError(err error, opts ...ErrorOption) {
	m.errors = append(m.errors, err)
}

func TestMockTracer(t *testing.T) {
	mock := &mockTracer{}
	ctx := context.Background()

	ctx, span := mock.Start(ctx, "test-operation",
		WithAttributes(String("service", "test")),
	)

	zhtest.AssertEqual(t, 1, len(mock.spans))

	mockSpan := mock.spans[0]
	zhtest.AssertEqual(t, "test-operation", mockSpan.name)
	zhtest.AssertEqual(t, 1, len(mockSpan.attributes))

	// Test span methods
	span.SetStatus(CodeOk, "success")
	zhtest.AssertEqual(t, CodeOk, mockSpan.statusCode)

	span.End()
	zhtest.AssertTrue(t, mockSpan.ended)

	// Verify context has span
	zhtest.AssertNotNil(t, SpanFromContext(ctx))
}

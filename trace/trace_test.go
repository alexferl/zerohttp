package trace

import (
	"context"
	"errors"
	"testing"
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
			if tt.attr.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", tt.attr.Key, tt.wantKey)
			}
			if tt.attr.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.wantVal)
			}
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
	if ctx == nil {
		t.Error("Expected non-nil context")
	}
}

func TestContextWithSpan(t *testing.T) {
	tracer := NewNoopTracer()

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test")

	// Store span in context
	ctx = ContextWithSpan(ctx, span)

	// Retrieve span from context
	retrieved := SpanFromContext(ctx)
	if retrieved == nil {
		t.Error("Expected to retrieve span from context")
	}

	// Nil context should return nil
	if SpanFromContext(context.TODO()) != nil {
		t.Error("Expected nil from context without span")
	}

	// Context without span should return nil
	emptyCtx := context.Background()
	if SpanFromContext(emptyCtx) != nil {
		t.Error("Expected nil from context without span")
	}
}

func TestSpanConfig(t *testing.T) {
	cfg := &spanConfig{}

	// Apply WithAttributes
	opt := WithAttributes(
		String("key1", "val1"),
		Int("key2", 42),
	)
	opt.apply(cfg)

	if len(cfg.attributes) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(cfg.attributes))
	}
}

func TestErrorConfig(t *testing.T) {
	cfg := &errorConfig{}

	// Apply WithErrorAttributes
	opt := WithErrorAttributes(String("error.type", "test"))
	opt.applyError(cfg)

	if len(cfg.attributes) != 1 {
		t.Errorf("Expected 1 attribute, got %d", len(cfg.attributes))
	}
}

func TestCodeConstants(t *testing.T) {
	if CodeUnset != 0 {
		t.Errorf("CodeUnset = %d, want 0", CodeUnset)
	}
	if CodeOk != 1 {
		t.Errorf("CodeOk = %d, want 1", CodeOk)
	}
	if CodeError != 2 {
		t.Errorf("CodeError = %d, want 2", CodeError)
	}
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

	if len(mock.spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(mock.spans))
	}

	mockSpan := mock.spans[0]
	if mockSpan.name != "test-operation" {
		t.Errorf("Name = %q, want %q", mockSpan.name, "test-operation")
	}

	if len(mockSpan.attributes) != 1 {
		t.Errorf("Expected 1 attribute, got %d", len(mockSpan.attributes))
	}

	// Test span methods
	span.SetStatus(CodeOk, "success")
	if mockSpan.statusCode != CodeOk {
		t.Errorf("Status code = %d, want %d", mockSpan.statusCode, CodeOk)
	}

	span.End()
	if !mockSpan.ended {
		t.Error("Expected span to be ended")
	}

	// Verify context has span
	if SpanFromContext(ctx) == nil {
		t.Error("Expected span in context")
	}
}

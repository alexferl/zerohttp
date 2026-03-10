package middleware

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/rwutil"
	"github.com/alexferl/zerohttp/trace"
)

// mockTracer is a test tracer that records span creation
type mockTracer struct {
	spans []*mockSpan
}

func (m *mockTracer) Start(ctx context.Context, name string, opts ...trace.SpanOption) (context.Context, trace.Span) {
	span := &mockSpan{name: name}
	m.spans = append(m.spans, span)

	// Capture initial attributes from options
	for _, opt := range opts {
		// We can't call apply directly since it's unexported,
		// but we can verify options work through integration tests
		_ = opt
	}

	return trace.ContextWithSpan(ctx, span), span
}

type mockSpan struct {
	name       string
	statusCode trace.Code
	statusDesc string
	attributes []trace.Attribute
	errors     []error
	ended      bool
}

func (m *mockSpan) End() { m.ended = true }
func (m *mockSpan) SetStatus(code trace.Code, description string) {
	m.statusCode = code
	m.statusDesc = description
}

func (m *mockSpan) SetAttributes(attrs ...trace.Attribute) {
	m.attributes = append(m.attributes, attrs...)
}

func (m *mockSpan) RecordError(err error, opts ...trace.ErrorOption) {
	m.errors = append(m.errors, err)
}

func TestTracing_CreatesSpan(t *testing.T) {
	mock := &mockTracer{}
	mw := Tracing(mock)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if len(mock.spans) != 1 {
		t.Fatalf("Expected 1 span, got %d", len(mock.spans))
	}

	span := mock.spans[0]
	if !span.ended {
		t.Error("Expected span to be ended")
	}
}

func TestTracing_SetsAttributes(t *testing.T) {
	mock := &mockTracer{}
	mw := Tracing(mock)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	span := mock.spans[0]
	attrs := make(map[string]any)
	for _, attr := range span.attributes {
		attrs[attr.Key] = attr.Value
	}

	// The status code is set via SetAttributes after Start
	if attrs["http.status_code"] != 201 {
		t.Errorf("status_code = %v, want 201", attrs["http.status_code"])
	}
}

func TestTracing_ContentLength(t *testing.T) {
	mock := &mockTracer{}
	mw := Tracing(mock)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request with body (ContentLength > 0)
	body := "test body"
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	span := mock.spans[0]
	attrs := make(map[string]any)
	for _, attr := range span.attributes {
		attrs[attr.Key] = attr.Value
	}

	if attrs["http.request_content_length"] != int64(9) {
		t.Errorf("request_content_length = %v, want 9", attrs["http.request_content_length"])
	}
}

func TestTracing_ExemptPaths(t *testing.T) {
	mock := &mockTracer{}
	mw := Tracing(mock, config.TracerConfig{
		ExemptPaths: []string{"/health", "/metrics"},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request to exempt path should not create span
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(mock.spans) != 0 {
		t.Errorf("Expected 0 spans for exempt path, got %d", len(mock.spans))
	}

	// Request to non-exempt path should create span
	req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(mock.spans) != 1 {
		t.Errorf("Expected 1 span for non-exempt path, got %d", len(mock.spans))
	}
}

func TestTracing_NilTracer(t *testing.T) {
	mw := Tracing(nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic with nil tracer
	handler.ServeHTTP(rr, req)
}

func TestTracing_StatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantCode   trace.Code
	}{
		{"success", 200, trace.CodeOk},
		{"created", 201, trace.CodeOk},
		{"bad_request", 400, trace.CodeOk},
		{"not_found", 404, trace.CodeOk},
		{"server_error", 500, trace.CodeError},
		{"bad_gateway", 502, trace.CodeError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockTracer{}
			mw := Tracing(mock)

			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			span := mock.spans[0]
			if span.statusCode != tt.wantCode {
				t.Errorf("statusCode = %d, want %d", span.statusCode, tt.wantCode)
			}
		})
	}
}

func TestTracing_SpanNameFormatter(t *testing.T) {
	mock := &mockTracer{}
	mw := Tracing(mock, config.TracerConfig{
		SpanNameFormatter: func(r *http.Request) string {
			return "custom:" + r.Method
		},
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	span := mock.spans[0]
	if span.name != "custom:GET" {
		t.Errorf("name = %q, want custom:GET", span.name)
	}
}

func TestScheme(t *testing.T) {
	tests := []struct {
		name       string
		tls        bool
		header     string
		wantScheme string
	}{
		{"http", false, "", "http"},
		{"https", true, "", "https"},
		{"forwarded_https", false, "https", "https"},
		{"forwarded_http", false, "http", "http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if tt.header != "" {
				req.Header.Set("X-Forwarded-Proto", tt.header)
			}

			got := scheme(req)
			if got != tt.wantScheme {
				t.Errorf("scheme() = %q, want %q", got, tt.wantScheme)
			}
		})
	}
}

func TestTracingResponseWriter(t *testing.T) {
	mock := &mockTracer{}
	ctx := context.Background()
	_, span := mock.Start(ctx, "test")

	rr := httptest.NewRecorder()
	rw := rwutil.NewResponseWriter(rr)
	trw := &tracingResponseWriter{
		ResponseWriter: rw,
		span:           span,
	}

	// Write should trigger WriteHeader
	_, _ = trw.Write([]byte("hello"))

	if trw.StatusCode() != 200 {
		t.Errorf("statusCode = %d, want 200", trw.StatusCode())
	}

	if rr.Code != 200 {
		t.Errorf("recorder code = %d, want 200", rr.Code)
	}

	// Multiple WriteHeader calls should be safe
	trw.WriteHeader(500)
	if trw.StatusCode() != 200 {
		t.Error("WriteHeader should not change status after already written")
	}
}

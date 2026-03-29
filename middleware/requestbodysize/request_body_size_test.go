package requestbodysize

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

type requestBodySizeTestHandler struct {
	called    bool
	bodyRead  []byte
	bodyError error
}

func (h *requestBodySizeTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	body, err := io.ReadAll(r.Body)
	h.bodyRead = body
	h.bodyError = err
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func TestRequestBodySize_Limits(t *testing.T) {
	tests := []struct {
		name        string
		maxBytes    int64
		bodyContent string
		expectError bool
	}{
		{"exceeds limit", 10, "this body is definitely longer than 10 bytes", true},
		{"exact limit", 10, "1234567890", false},
		{"under limit", 20, "short", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := New(Config{MaxBytes: tt.maxBytes})(handler)
			body := bytes.NewReader([]byte(tt.bodyContent))
			req := zhtest.NewRequest(http.MethodPost, "/").WithBody(body).Build()
			zhtest.Serve(middleware, req)

			zhtest.AssertTrue(t, handler.called)
			hasError := handler.bodyError != nil
			zhtest.AssertEqual(t, tt.expectError, hasError)
			if tt.expectError && hasError {
				zhtest.AssertTrue(t, strings.Contains(handler.bodyError.Error(), "too large"))
			}
		})
	}
}

func TestRequestBodySize_ExcludedPaths(t *testing.T) {
	tests := []struct {
		path        string
		expectError bool
	}{
		{"/upload", false},
		{"/webhook", false},
		{"/api/large", false},
		{"/api/small", true},
		{"/other", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := New(Config{
				MaxBytes:      5,
				ExcludedPaths: []string{"/upload", "/webhook", "/api/large"},
			})(handler)
			largeBody := bytes.NewReader([]byte("this is a long body"))
			req := zhtest.NewRequest(http.MethodPost, tt.path).WithBody(largeBody).Build()
			zhtest.Serve(middleware, req)

			zhtest.AssertTrue(t, handler.called)
			hasError := handler.bodyError != nil
			if !tt.expectError && hasError {
				zhtest.AssertFailf(t, "Expected no error for excluded path %s, got %v", tt.path, handler.bodyError)
			}
			if tt.expectError && !hasError {
				zhtest.AssertFailf(t, "Expected error for non-excluded path %s, got none", tt.path)
			}
		})
	}
}

func TestRequestBodySize_EmptyExcludedPaths(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := New(Config{
		MaxBytes:      10,
		ExcludedPaths: []string{},
	})(handler)
	largeBody := bytes.NewReader([]byte("this body is longer than 10 bytes"))
	req := zhtest.NewRequest(http.MethodPost, "/any-path").WithBody(largeBody).Build()
	zhtest.Serve(middleware, req)

	zhtest.AssertTrue(t, handler.called)
	zhtest.AssertError(t, handler.bodyError)
}

func TestRequestBodySize_ConfigFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		maxBytes int64
	}{
		{"invalid negative", -1},
		{"zero max bytes", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := New(Config{MaxBytes: tt.maxBytes})(handler)
			smallBody := bytes.NewReader([]byte("small body"))
			req := zhtest.NewRequest(http.MethodPost, "/").WithBody(smallBody).Build()
			zhtest.Serve(middleware, req)

			zhtest.AssertTrue(t, handler.called)
			zhtest.AssertNoError(t, handler.bodyError)
		})
	}
}

func TestRequestBodySize_NilExcludedPaths(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := New(Config{
		MaxBytes:      100,
		ExcludedPaths: nil,
	})(handler)
	smallBody := bytes.NewReader([]byte("small body"))
	req := zhtest.NewRequest(http.MethodPost, "/").WithBody(smallBody).Build()
	zhtest.Serve(middleware, req)

	zhtest.AssertTrue(t, handler.called)
	zhtest.AssertNoError(t, handler.bodyError)
}

func TestRequestBodySize_MultipleOptions(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := New(Config{
		MaxBytes:      10,
		ExcludedPaths: []string{"/test"},
	})(handler)
	largeBody := bytes.NewReader([]byte("this body is longer than 10 bytes but less than 100"))
	req := zhtest.NewRequest(http.MethodPost, "/").WithBody(largeBody).Build()
	zhtest.Serve(middleware, req)

	zhtest.AssertTrue(t, handler.called)
	zhtest.AssertError(t, handler.bodyError)
}

func TestDefaultRequestBodySizeConfig(t *testing.T) {
	cfg := DefaultConfig
	expectedMaxBytes := int64(1 << 20)
	zhtest.AssertEqual(t, expectedMaxBytes, cfg.MaxBytes)
	zhtest.AssertNotNil(t, cfg.ExcludedPaths)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
}

func TestRequestBodySize_GetRequest(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := New(Config{MaxBytes: 10})(handler)
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(middleware, req)

	zhtest.AssertTrue(t, handler.called)
	zhtest.AssertNoError(t, handler.bodyError)
}

func TestRequestBodySize_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := New(Config{MaxBytes: 10})

	metricsMw := metrics.NewMiddleware(reg, metrics.Config{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	wrapped := metricsMw(handler)

	// Test request within limit
	req1 := zhtest.NewRequest(http.MethodPost, "/").WithBody(strings.NewReader("short")).Build()
	zhtest.Serve(wrapped, req1)

	// Test request exceeding limit
	req2 := zhtest.NewRequest(http.MethodPost, "/").WithBody(strings.NewReader("this is definitely longer than 10 bytes")).Build()
	zhtest.Serve(wrapped, req2)

	// Check metrics
	families := reg.Gather()

	var rejectedCounter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "request_body_size_rejected_total" {
			rejectedCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, rejectedCounter)

	rejected := 0
	for _, m := range rejectedCounter.Metrics {
		rejected = int(m.Counter)
	}
	zhtest.AssertEqual(t, 1, rejected)
}

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestRequestBodySize_Flush(t *testing.T) {
	tests := []struct {
		name              string
		underlyingFlusher bool
		expectFlushCalled bool
	}{
		{
			name:              "flush passes through to underlying Flusher",
			underlyingFlusher: true,
			expectFlushCalled: true,
		},
		{
			name:              "flush no-op when underlying doesn't implement Flusher",
			underlyingFlusher: false,
			expectFlushCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base http.ResponseWriter
			var flushCalled *bool

			if tt.underlyingFlusher {
				rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
				base = rec
				flushCalled = &rec.flushed
			} else {
				rec := httptest.NewRecorder()
				base = rec
				flushCalled = new(bool) // always false
			}

			// Wrap with limitResponseWriter
			lrw := &limitResponseWriter{
				ResponseWriter: base,
			}

			// Call Flush
			lrw.Flush()

			zhtest.AssertEqual(t, tt.expectFlushCalled, *flushCalled)
		})
	}
}

func TestRequestBodySize_Flush_SupportsSSE(t *testing.T) {
	// This test verifies that SSE streaming works through the middleware
	rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}

	mw := New(Config{MaxBytes: 1024})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get a Flusher from the writer
		f, ok := w.(http.Flusher)
		zhtest.AssertTrue(t, ok)

		// Write and flush like SSE would
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextEventStream)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
		f.Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	handler.ServeHTTP(rec, req)

	zhtest.AssertTrue(t, rec.flushed)
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestRequestBodySize_IncludedPaths(t *testing.T) {
	tests := []struct {
		path        string
		expectError bool
	}{
		{"/api/upload", true},
		{"/api/data", true},
		{"/admin", true},
		{"/health", false},
		{"/metrics", false},
		{"/public", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := New(Config{
				MaxBytes:      5,
				IncludedPaths: []string{"/api/", "/admin"},
			})(handler)
			largeBody := bytes.NewReader([]byte("this is a long body"))
			req := zhtest.NewRequest(http.MethodPost, tt.path).WithBody(largeBody).Build()
			zhtest.Serve(middleware, req)

			zhtest.AssertTrue(t, handler.called)
			hasError := handler.bodyError != nil
			if !tt.expectError && hasError {
				zhtest.AssertFailf(t, "Expected no error for non-allowed path %s, got %v", tt.path, handler.bodyError)
			}
			if tt.expectError && !hasError {
				zhtest.AssertFailf(t, "Expected error for allowed path %s, got none", tt.path)
			}
		})
	}
}

func TestRequestBodySize_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			zhtest.AssertFail(t, "expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = New(Config{
		MaxBytes:      1024,
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api"},
	})
}

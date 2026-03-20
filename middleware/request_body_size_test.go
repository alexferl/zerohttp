package middleware

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
			middleware := RequestBodySize(config.RequestBodySizeConfig{MaxBytes: tt.maxBytes})(handler)
			body := bytes.NewReader([]byte(tt.bodyContent))
			req := zhtest.NewRequest(http.MethodPost, "/").WithBody(body).Build()
			zhtest.Serve(middleware, req)

			if !handler.called {
				t.Error("Expected handler to be called")
			}
			hasError := handler.bodyError != nil
			if tt.expectError && !hasError {
				t.Error("Expected error when body exceeds limit")
			}
			if !tt.expectError && hasError {
				t.Errorf("Expected no error, got %v", handler.bodyError)
			}
			if tt.expectError && hasError && !strings.Contains(handler.bodyError.Error(), "too large") {
				t.Errorf("Expected 'too large' error, got: %v", handler.bodyError)
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
			middleware := RequestBodySize(config.RequestBodySizeConfig{
				MaxBytes:      5,
				ExcludedPaths: []string{"/upload", "/webhook", "/api/large"},
			})(handler)
			largeBody := bytes.NewReader([]byte("this is a long body"))
			req := zhtest.NewRequest(http.MethodPost, tt.path).WithBody(largeBody).Build()
			zhtest.Serve(middleware, req)

			if !handler.called {
				t.Errorf("Expected handler to be called for path %s", tt.path)
			}
			hasError := handler.bodyError != nil
			if !tt.expectError && hasError {
				t.Errorf("Expected no error for excluded path %s, got %v", tt.path, handler.bodyError)
			}
			if tt.expectError && !hasError {
				t.Errorf("Expected error for non-excluded path %s, got none", tt.path)
			}
		})
	}
}

func TestRequestBodySize_EmptyExcludedPaths(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(config.RequestBodySizeConfig{
		MaxBytes:      10,
		ExcludedPaths: []string{},
	})(handler)
	largeBody := bytes.NewReader([]byte("this body is longer than 10 bytes"))
	req := zhtest.NewRequest(http.MethodPost, "/any-path").WithBody(largeBody).Build()
	zhtest.Serve(middleware, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError == nil {
		t.Error("Expected error when no paths are excluded")
	}
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
			middleware := RequestBodySize(config.RequestBodySizeConfig{MaxBytes: tt.maxBytes})(handler)
			smallBody := bytes.NewReader([]byte("small body"))
			req := zhtest.NewRequest(http.MethodPost, "/").WithBody(smallBody).Build()
			zhtest.Serve(middleware, req)

			if !handler.called {
				t.Error("Expected handler to be called")
			}
			if handler.bodyError != nil {
				t.Errorf("Expected no error with default config, got %v", handler.bodyError)
			}
		})
	}
}

func TestRequestBodySize_NilExcludedPaths(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(config.RequestBodySizeConfig{
		MaxBytes:      100,
		ExcludedPaths: nil,
	})(handler)
	smallBody := bytes.NewReader([]byte("small body"))
	req := zhtest.NewRequest(http.MethodPost, "/").WithBody(smallBody).Build()
	zhtest.Serve(middleware, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError != nil {
		t.Errorf("Expected no error with nil excluded paths, got %v", handler.bodyError)
	}
}

func TestRequestBodySize_MultipleOptions(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(config.RequestBodySizeConfig{
		MaxBytes:      10,
		ExcludedPaths: []string{"/test"},
	})(handler)
	largeBody := bytes.NewReader([]byte("this body is longer than 10 bytes but less than 100"))
	req := zhtest.NewRequest(http.MethodPost, "/").WithBody(largeBody).Build()
	zhtest.Serve(middleware, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError == nil {
		t.Error("Expected error with 10 byte limit")
	}
}

func TestDefaultRequestBodySizeConfig(t *testing.T) {
	cfg := config.DefaultRequestBodySizeConfig
	expectedMaxBytes := int64(1 << 20)
	if cfg.MaxBytes != expectedMaxBytes {
		t.Errorf("Expected default max bytes %d, got %d", expectedMaxBytes, cfg.MaxBytes)
	}
	if cfg.ExcludedPaths == nil {
		t.Error("Expected default excluded paths to be empty slice, got nil")
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("Expected default excluded paths to be empty, got %d items", len(cfg.ExcludedPaths))
	}
}

func TestRequestBodySize_GetRequest(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(config.RequestBodySizeConfig{MaxBytes: 10})(handler)
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(middleware, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError != nil {
		t.Errorf("Expected no error for GET request, got %v", handler.bodyError)
	}
}

func TestRequestBodySize_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := RequestBodySize(config.RequestBodySizeConfig{MaxBytes: 10})

	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
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

	if rejectedCounter == nil {
		t.Fatal("expected request_body_size_rejected_total metric")
	}

	rejected := 0
	for _, m := range rejectedCounter.Metrics {
		rejected = int(m.Counter)
	}
	if rejected != 1 {
		t.Errorf("expected 1 rejected request, got %d", rejected)
	}
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

			if *flushCalled != tt.expectFlushCalled {
				t.Errorf("expected flush called=%v, got=%v", tt.expectFlushCalled, *flushCalled)
			}
		})
	}
}

func TestRequestBodySize_Flush_SupportsSSE(t *testing.T) {
	// This test verifies that SSE streaming works through the middleware
	rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}

	mw := RequestBodySize(config.RequestBodySizeConfig{MaxBytes: 1024})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get a Flusher from the writer
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("expected ResponseWriter to implement http.Flusher")
			return
		}

		// Write and flush like SSE would
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextEventStream)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
		f.Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	handler.ServeHTTP(rec, req)

	if !rec.flushed {
		t.Error("expected Flush to be called on underlying ResponseWriter")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
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
			middleware := RequestBodySize(config.RequestBodySizeConfig{
				MaxBytes:      5,
				IncludedPaths: []string{"/api/", "/admin"},
			})(handler)
			largeBody := bytes.NewReader([]byte("this is a long body"))
			req := zhtest.NewRequest(http.MethodPost, tt.path).WithBody(largeBody).Build()
			zhtest.Serve(middleware, req)

			if !handler.called {
				t.Errorf("Expected handler to be called for path %s", tt.path)
			}
			hasError := handler.bodyError != nil
			if !tt.expectError && hasError {
				t.Errorf("Expected no error for non-allowed path %s, got %v", tt.path, handler.bodyError)
			}
			if tt.expectError && !hasError {
				t.Errorf("Expected error for allowed path %s, got none", tt.path)
			}
		})
	}
}

func TestRequestBodySize_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when both ExcludedPaths and IncludedPaths are set")
		}
	}()

	_ = RequestBodySize(config.RequestBodySizeConfig{
		MaxBytes:      1024,
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api"},
	})
}

package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

// flusherRecorder is a test ResponseWriter that implements http.Flusher
type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestNewMiddleware_NoConfig(t *testing.T) {
	// Test calling NewMiddleware with no config (uses defaults)
	// When a registry is passed, metrics should be recorded
	reg := NewRegistry()
	middleware := NewMiddleware(reg)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	zhtest.AssertTrue(t, called)
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Metrics should be recorded since we passed a registry
	families := reg.Gather()
	zhtest.AssertGreater(t, len(families), 0)
}

func TestNewMiddleware_NilRegistry(t *testing.T) {
	// Nil registry means metrics disabled - middleware should pass through
	middleware := NewMiddleware(nil)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	zhtest.AssertTrue(t, called)
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestMiddleware_BasicRequest(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		Endpoint:        "/metrics",
		DurationBuckets: []float64{0.001, 0.01, 0.1, 1},
		SizeBuckets:     []float64{100, 1000, 10000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Check that metrics were recorded
	families := reg.Gather()

	metricNames := make(map[string]bool)
	for _, f := range families {
		metricNames[f.Name] = true
	}

	expectedMetrics := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_response_size_bytes",
		"http_requests_in_flight",
	}

	for _, name := range expectedMetrics {
		zhtest.AssertTrue(t, metricNames[name])
	}
}

func TestMiddleware_ExcludedPath(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		ExcludedPaths: []string{"/health", "/metrics"},
		PathLabelFunc: func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Request to excluded path
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Check that no metrics values were recorded for excluded path
	families := reg.Gather()

	// The metrics exist but should have no values for the excluded path
	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	if requestCounter != nil {
		for _, m := range requestCounter.Metrics {
			zhtest.AssertNotEqual(t, "/health", m.Labels["path"])
		}
	}
}

func TestMiddleware_DifferentStatusCodes(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001, 0.01, 0.1},
		SizeBuckets:     []float64{100, 1000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	// Handler that returns different status codes
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		switch status {
		case "200":
			w.WriteHeader(http.StatusOK)
		case "404":
			w.WriteHeader(http.StatusNotFound)
		case "500":
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	wrapped := middleware(handler)

	// Make requests with different status codes
	for _, code := range []string{"200", "404", "500"} {
		req := httptest.NewRequest(http.MethodGet, "/test?status="+code, nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}

	// Check that metrics have different status labels
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	statuses := make(map[string]int)
	for _, m := range requestCounter.Metrics {
		if status, ok := m.Labels["status"]; ok {
			statuses[status]++
		}
	}

	zhtest.AssertEqual(t, 3, len(statuses))
}

func TestMiddleware_RequestSize(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001, 0.01},
		SizeBuckets:     []float64{10, 100, 1000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Request with body
	body := strings.NewReader("test body content")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(body.Len())
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check that request size metric exists
	families := reg.Gather()

	var found bool
	for _, f := range families {
		if f.Name == "http_request_size_bytes" {
			found = true
			break
		}
	}

	zhtest.AssertTrue(t, found)
}

func TestMiddleware_ResponseSize(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001, 0.01},
		SizeBuckets:     []float64{10, 100, 1000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	responseBody := "this is a test response"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(responseBody))
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check response size was recorded
	families := reg.Gather()

	var responseSizeHist *MetricFamily
	for _, f := range families {
		if f.Name == "http_response_size_bytes" {
			responseSizeHist = &f
			break
		}
	}

	zhtest.AssertNotNil(t, responseSizeHist)
	// Should have at least one metric
	zhtest.AssertGreater(t, len(responseSizeHist.Metrics), 0)
}

func TestMiddleware_InFlightGauge(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	done := make(chan bool)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow request
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		done <- true
	})

	wrapped := middleware(handler)

	// Start a slow request
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}()

	// Wait a bit for request to start
	time.Sleep(10 * time.Millisecond)

	// Check in-flight gauge is incremented
	families := reg.Gather()

	var inFlight *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_in_flight" {
			inFlight = &f
			break
		}
	}

	zhtest.AssertNotNil(t, inFlight)

	// Wait for request to complete
	<-done
}

func TestMiddleware_CustomPathLabelFunc(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc: func(p string) string {
			// Normalize IDs in paths
			if strings.HasPrefix(p, "/users/") {
				return "/users/{id}"
			}
			return p
		},
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Request to path with ID
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Check that path label is normalized
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	found := false
	for _, m := range requestCounter.Metrics {
		if m.Labels["path"] == "/users/{id}" {
			found = true
			break
		}
	}

	zhtest.AssertTrue(t, found)
}

func TestMiddleware_DefaultStatusCode(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	// Handler that doesn't explicitly set status code
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
		// Status 200 is set implicitly by Write if not already set
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check that default status (200) is recorded
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	found := false
	for _, m := range requestCounter.Metrics {
		if m.Labels["status"] == "200" {
			found = true
			break
		}
	}

	zhtest.AssertTrue(t, found)
}

func TestResponseWriter_CaptureStatusAndSize(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: base,
		statusCode:     0,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	zhtest.AssertEqual(t, http.StatusCreated, rw.statusCode)

	// Test that second WriteHeader doesn't change status
	rw.WriteHeader(http.StatusInternalServerError)
	zhtest.AssertEqual(t, http.StatusCreated, rw.statusCode)

	// Test Write
	data := []byte("hello world")
	n, err := rw.Write(data)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, len(data), n)
	zhtest.AssertEqual(t, int64(len(data)), rw.size)
}

func TestResponseWriter_WriteSetsDefaultStatus(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: base,
		statusCode:     0,
	}

	// Write without explicit WriteHeader should set 200
	_, _ = rw.Write([]byte("test"))

	zhtest.AssertEqual(t, http.StatusOK, rw.statusCode)
}

func TestMiddleware_MultipleMethods(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Make requests with different methods
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}

	// Check that metrics have different method labels
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	methodsFound := make(map[string]bool)
	for _, m := range requestCounter.Metrics {
		if method, ok := m.Labels["method"]; ok {
			methodsFound[method] = true
		}
	}

	for _, method := range methods {
		zhtest.AssertTrue(t, methodsFound[method])
	}
}

func TestMiddleware_Router404And405(t *testing.T) {
	// Create a real zerohttp router to test 404/405 handling
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001, 0.01, 0.1},
		SizeBuckets:     []float64{100, 1000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	// Create a router-like handler that returns 405 for wrong method
	// and 404 for unknown paths
	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/exists" {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// Wrap router with middleware
	wrapped := middleware(router)

	// Test 200
	req1 := httptest.NewRequest(http.MethodGet, "/exists", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)
	zhtest.AssertEqual(t, http.StatusOK, rec1.Code)

	// Test 405
	req2 := httptest.NewRequest(http.MethodPost, "/exists", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)
	zhtest.AssertEqual(t, http.StatusMethodNotAllowed, rec2.Code)

	// Test 404
	req3 := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec3 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec3, req3)
	zhtest.AssertEqual(t, http.StatusNotFound, rec3.Code)

	// Check that all status codes are recorded
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	statuses := make(map[string]uint64)
	for _, m := range requestCounter.Metrics {
		if status, ok := m.Labels["status"]; ok {
			statuses[status] = m.Counter
		}
	}

	zhtest.AssertEqual(t, uint64(1), statuses["200"])
	zhtest.AssertEqual(t, uint64(1), statuses["404"])
	zhtest.AssertEqual(t, uint64(1), statuses["405"])
}

func TestMiddleware_NotFound(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	// Handler that only handles GET
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Make a GET request (should be 200)
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	// Make a POST request (should be 405)
	req2 := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	// Check that both status codes are recorded
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	statuses := make(map[string]uint64)
	for _, m := range requestCounter.Metrics {
		if status, ok := m.Labels["status"]; ok {
			statuses[status] = m.Counter
		}
	}

	zhtest.AssertEqual(t, uint64(1), statuses["200"])
	zhtest.AssertEqual(t, uint64(1), statuses["405"])
}

func TestMiddleware_CustomLabels(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc:   func(p string) string { return p },
		CustomLabels: func(r *http.Request) map[string]string {
			return map[string]string{
				"tenant": r.Header.Get("X-Tenant-ID"),
				"region": "us-east",
			}
		},
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Make request with custom tenant header
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Check that metrics have custom labels
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)
	zhtest.AssertEqual(t, 1, len(requestCounter.Metrics))

	m := requestCounter.Metrics[0]

	// Check standard labels
	zhtest.AssertEqual(t, "GET", m.Labels["method"])
	zhtest.AssertEqual(t, "200", m.Labels["status"])

	// Check custom labels
	zhtest.AssertEqual(t, "tenant-123", m.Labels["tenant"])
	zhtest.AssertEqual(t, "us-east", m.Labels["region"])
}

func TestMiddleware_PanicRecords500(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001, 0.01, 0.1},
		SizeBuckets:     []float64{100, 1000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	// Handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("intentional panic for testing")
	})

	wrapped := middleware(handler)

	// Make request that will panic
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	// Recover from the panic that will be re-panicked
	func() {
		defer func() {
			_ = recover() // Ignore the re-panic
		}()
		wrapped.ServeHTTP(rec, req)
	}()

	// Check that metrics recorded status 500
	families := reg.Gather()

	var requestCounter *MetricFamily
	for _, f := range families {
		if f.Name == "http_requests_total" {
			requestCounter = &f
			break
		}
	}

	zhtest.AssertNotNil(t, requestCounter)

	found500 := false
	for _, m := range requestCounter.Metrics {
		if m.Labels["status"] == "500" {
			found500 = true
			break
		}
	}

	zhtest.AssertTrue(t, found500)
}

func TestMiddleware_RegistryInContext(t *testing.T) {
	reg := NewRegistry()
	cfg := Config{
		DurationBuckets: []float64{0.001},
		SizeBuckets:     []float64{100},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	var ctxRegistry Registry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get registry from context
		ctxRegistry = GetRegistry(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	zhtest.AssertNotNil(t, ctxRegistry)
	zhtest.AssertEqual(t, reg, ctxRegistry)
}

func TestMiddleware_responseWriter_Flush(t *testing.T) {
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
				flushCalled = new(bool)
			}

			// Wrap with responseWriter
			rw := &responseWriter{
				ResponseWriter: base,
			}

			// Call Flush
			rw.Flush()

			zhtest.AssertEqual(t, tt.expectFlushCalled, *flushCalled)
		})
	}
}

func TestMiddleware_responseWriter_Flush_SupportsSSE(t *testing.T) {
	rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}

	reg := NewRegistry()
	middleware := NewMiddleware(reg, Config{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

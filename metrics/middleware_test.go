package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
)

func TestNewMiddleware_Disabled(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{Enabled: false}

	middleware := NewMiddleware(reg, cfg)

	// Middleware should return the handler as-is when disabled
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should have been called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewMiddleware_NilRegistry(t *testing.T) {
	cfg := config.MetricsConfig{Enabled: true}

	middleware := NewMiddleware(nil, cfg)

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should have been called")
	}
}

func TestMiddleware_BasicRequest(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

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
		if !metricNames[name] {
			t.Errorf("expected metric %s to be recorded", name)
		}
	}
}

func TestMiddleware_ExcludedPath(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:       true,
		ExcludePaths:  []string{"/health", "/metrics"},
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

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

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
			if m.Labels["path"] == "/health" {
				t.Error("expected no metrics for excluded /health path")
			}
		}
	}
}

func TestMiddleware_DifferentStatusCodes(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if requestCounter == nil {
		t.Fatal("expected http_requests_total metric")
	}

	statuses := make(map[string]int)
	for _, m := range requestCounter.Metrics {
		if status, ok := m.Labels["status"]; ok {
			statuses[status]++
		}
	}

	if len(statuses) != 3 {
		t.Errorf("expected 3 different status codes, got %d: %v", len(statuses), statuses)
	}
}

func TestMiddleware_RequestSize(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if !found {
		t.Error("expected http_request_size_bytes metric")
	}
}

func TestMiddleware_ResponseSize(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if responseSizeHist == nil {
		t.Fatal("expected http_response_size_bytes metric")
	}

	// Should have at least one metric
	if len(responseSizeHist.Metrics) == 0 {
		t.Error("expected at least one histogram metric")
	}
}

func TestMiddleware_InFlightGauge(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if inFlight == nil {
		t.Fatal("expected http_requests_in_flight metric")
	}

	// Wait for request to complete
	<-done
}

func TestMiddleware_CustomPathLabelFunc(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if requestCounter == nil {
		t.Fatal("expected http_requests_total metric")
	}

	found := false
	for _, m := range requestCounter.Metrics {
		if m.Labels["path"] == "/users/{id}" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected path label to be normalized to /users/{id}")
	}
}

func TestMiddleware_DefaultStatusCode(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if requestCounter == nil {
		t.Fatal("expected http_requests_total metric")
	}

	found := false
	for _, m := range requestCounter.Metrics {
		if m.Labels["status"] == "200" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected status 200 to be recorded for implicit OK response")
	}
}

func TestResponseWriter_CaptureStatusAndSize(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: base,
		statusCode:     0,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rw.statusCode)
	}

	// Test that second WriteHeader doesn't change status
	rw.WriteHeader(http.StatusInternalServerError)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("status should not change after first WriteHeader")
	}

	// Test Write
	data := []byte("hello world")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}
	if rw.size != int64(len(data)) {
		t.Errorf("expected size %d, got %d", len(data), rw.size)
	}
}

func TestResponseWriter_WriteSetsDefaultStatus(t *testing.T) {
	base := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: base,
		statusCode:     0,
	}

	// Write without explicit WriteHeader should set 200
	_, _ = rw.Write([]byte("test"))

	if rw.statusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rw.statusCode)
	}
}

func TestMiddleware_MultipleMethods(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if requestCounter == nil {
		t.Fatal("expected http_requests_total metric")
	}

	methodsFound := make(map[string]bool)
	for _, m := range requestCounter.Metrics {
		if method, ok := m.Labels["method"]; ok {
			methodsFound[method] = true
		}
	}

	for _, method := range methods {
		if !methodsFound[method] {
			t.Errorf("expected method %s to be recorded", method)
		}
	}
}

func BenchmarkMiddleware(b *testing.B) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
		DurationBuckets: []float64{0.001, 0.01, 0.1},
		SizeBuckets:     []float64{100, 1000},
		PathLabelFunc:   func(p string) string { return p },
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddleware_CustomLabels(b *testing.B) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
		DurationBuckets: []float64{0.001, 0.01, 0.1},
		SizeBuckets:     []float64{100, 1000},
		PathLabelFunc:   func(p string) string { return p },
		CustomLabels: func(r *http.Request) map[string]string {
			return map[string]string{
				"tenant": "tenant-123",
				"region": "us-east",
			}
		},
	}

	middleware := NewMiddleware(reg, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}

func TestMiddleware_CustomLabels(t *testing.T) {
	reg := NewRegistry()
	cfg := config.MetricsConfig{
		Enabled:         true,
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

	if requestCounter == nil {
		t.Fatal("expected http_requests_total metric")
	}

	if len(requestCounter.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(requestCounter.Metrics))
	}

	m := requestCounter.Metrics[0]

	// Check standard labels
	if m.Labels["method"] != "GET" {
		t.Errorf("expected method=GET, got %s", m.Labels["method"])
	}
	if m.Labels["status"] != "200" {
		t.Errorf("expected status=200, got %s", m.Labels["status"])
	}

	// Check custom labels
	if m.Labels["tenant"] != "tenant-123" {
		t.Errorf("expected tenant=tenant-123, got %s", m.Labels["tenant"])
	}
	if m.Labels["region"] != "us-east" {
		t.Errorf("expected region=us-east, got %s", m.Labels["region"])
	}
}

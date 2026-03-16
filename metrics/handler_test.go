package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

func TestHandler_NilRegistry(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "metrics not enabled") {
		t.Errorf("expected error message, got: %s", body)
	}
}

func TestHandler_EmptyRegistry(t *testing.T) {
	reg := NewRegistry()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get(httpx.HeaderContentType)
	if contentType != httpx.MIMETextPlainCharset {
		t.Errorf("expected text/plain content type, got: %s", contentType)
	}
}

func TestHandler_Counter(t *testing.T) {
	reg := NewRegistry()
	counter := reg.Counter("test_counter", "label")
	counter.WithLabelValues("value1").Inc()
	counter.WithLabelValues("value2").Add(5)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "# HELP test_counter Total test_counter") {
		t.Error("expected HELP line for test_counter")
	}

	if !strings.Contains(body, "# TYPE test_counter counter") {
		t.Error("expected TYPE line for test_counter")
	}

	if !strings.Contains(body, `test_counter{label="value1"} 1`) {
		t.Error("expected counter value1=1")
	}

	if !strings.Contains(body, `test_counter{label="value2"} 5`) {
		t.Error("expected counter value2=5")
	}
}

func TestHandler_Gauge(t *testing.T) {
	reg := NewRegistry()
	gauge := reg.Gauge("test_gauge", "label")
	gauge.WithLabelValues("a").Set(42.5)
	gauge.WithLabelValues("b").Inc()
	gauge.WithLabelValues("b").Add(2.5)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "# TYPE test_gauge gauge") {
		t.Error("expected TYPE line for test_gauge")
	}

	if !strings.Contains(body, `test_gauge{label="a"} 42.5`) {
		t.Error("expected gauge a=42.5")
	}

	if !strings.Contains(body, `test_gauge{label="b"} 3.5`) {
		t.Errorf("expected gauge b=3.5, got body:\n%s", body)
	}
}

func TestHandler_Histogram(t *testing.T) {
	reg := NewRegistry()
	buckets := []float64{0.1, 0.5, 1.0}
	hist := reg.Histogram("test_histogram", buckets, "method")
	hist.WithLabelValues("GET").Observe(0.05)
	hist.WithLabelValues("GET").Observe(0.3)
	hist.WithLabelValues("GET").Observe(2.0)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "# TYPE test_histogram histogram") {
		t.Error("expected TYPE line for test_histogram")
	}

	// Check bucket lines exist
	if !strings.Contains(body, `test_histogram_bucket{method="GET",le="0.1"} 1`) {
		t.Error("expected bucket le=0.1 count=1")
	}

	if !strings.Contains(body, `test_histogram_bucket{method="GET",le="0.5"} 2`) {
		t.Error("expected bucket le=0.5 count=2")
	}

	if !strings.Contains(body, `test_histogram_bucket{method="GET",le="+Inf"} 3`) {
		t.Error("expected +Inf bucket with cumulative count")
	}

	if !strings.Contains(body, `test_histogram_sum{method="GET"} 2.35`) {
		t.Errorf("expected histogram sum, got body:\n%s", body)
	}

	if !strings.Contains(body, `test_histogram_count{method="GET"} 3`) {
		t.Error("expected histogram count=3")
	}
}

func TestHandler_NoLabels(t *testing.T) {
	reg := NewRegistry()
	counter := reg.Counter("simple_counter")
	counter.Inc()

	gauge := reg.Gauge("simple_gauge")
	gauge.Set(100)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Metrics without labels should not have curly braces
	if !strings.Contains(body, "simple_counter 1") {
		t.Error("expected simple_counter 1 without labels")
	}

	if !strings.Contains(body, "simple_gauge 100") {
		t.Error("expected simple_gauge 100 without labels")
	}
}

func TestHandler_LabelEscaping(t *testing.T) {
	reg := NewRegistry()
	counter := reg.Counter("escape_test", "label")
	counter.WithLabelValues(`value with "quotes"`).Inc()
	counter.WithLabelValues("value with \\ backslash").Inc()
	counter.WithLabelValues("value with \n newline").Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check escaping
	if !strings.Contains(body, `\"`) {
		t.Error("expected escaped quotes")
	}

	if !strings.Contains(body, `\\`) {
		t.Error("expected escaped backslash")
	}

	if !strings.Contains(body, `\n`) {
		t.Error("expected escaped newline")
	}
}

func TestHandler_MultipleLabelNames(t *testing.T) {
	reg := NewRegistry()
	counter := reg.Counter("multi_label", "method", "status", "path")
	counter.WithLabelValues("GET", "200", "/api/users").Inc()
	counter.WithLabelValues("POST", "201", "/api/orders").Add(3)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Labels should be sorted alphabetically
	if !strings.Contains(body, `method="GET"`) {
		t.Error("expected method label")
	}

	if !strings.Contains(body, `path="/api/users"`) {
		t.Error("expected path label")
	}

	if !strings.Contains(body, `status="200"`) {
		t.Error("expected status label")
	}
}

func TestHandler_SortedOutput(t *testing.T) {
	reg := NewRegistry()
	// Create metrics in non-alphabetical order
	reg.Counter("zebra", "x")
	reg.Counter("alpha", "y")
	reg.Gauge("middle", "z")

	// Add values
	reg.Counter("zebra", "x").WithLabelValues("a").Inc()
	reg.Counter("alpha", "y").WithLabelValues("b").Inc()
	reg.Gauge("middle", "z").WithLabelValues("c").Set(1)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler := Handler(reg)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check that alpha appears before middle, and middle before zebra
	alphaIdx := strings.Index(body, "alpha")
	middleIdx := strings.Index(body, "middle")
	zebraIdx := strings.Index(body, "zebra")

	if alphaIdx == -1 || middleIdx == -1 || zebraIdx == -1 {
		t.Fatal("could not find all metric names")
	}

	if alphaIdx >= middleIdx || middleIdx >= zebraIdx {
		t.Error("metrics not sorted alphabetically")
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "single label",
			labels:   map[string]string{"key": "value"},
			expected: `key="value"`,
		},
		{
			name:     "multiple labels sorted",
			labels:   map[string]string{"b": "2", "a": "1", "c": "3"},
			expected: `a="1",b="2",c="3"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabels(tt.labels)
			if result != tt.expected {
				t.Errorf("formatLabels() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestEscapeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{`with"quotes`, `with\"quotes`},
		{`with\backslash`, `with\\backslash`},
		{"with\nnewline", `with\nnewline`},
		{`complex"\`, `complex\"\\`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeLabel(tt.input)
			if result != tt.expected {
				t.Errorf("escapeLabel(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMetricKey(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "empty",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "single",
			labels:   map[string]string{"a": "1"},
			expected: "a=1",
		},
		{
			name:     "multiple sorted",
			labels:   map[string]string{"b": "2", "a": "1"},
			expected: "a=1,b=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metricKey(tt.labels)
			if result != tt.expected {
				t.Errorf("metricKey() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

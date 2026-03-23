package tracer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTracerConfigWrapper_IsExcluded(t *testing.T) {
	tests := []struct {
		name          string
		excludedPaths []string
		path          string
		want          bool
	}{
		{"exact match", []string{"/health", "/metrics"}, "/health", true},
		{"not excluded", []string{"/health", "/metrics"}, "/api/users", false},
		{"empty excluded list", []string{}, "/health", false},
		{"nil excluded list", nil, "/health", false},
		{"partial match should not work", []string{"/health"}, "/healthz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{ExcludedPaths: tt.excludedPaths}
			wrapper := cfg.Wrap()
			got := wrapper.IsExcluded(tt.path)
			if got != tt.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestTracerConfigWrapper_GetSpanName(t *testing.T) {
	tests := []struct {
		name      string
		formatter func(r *http.Request) string
		method    string
		path      string
		want      string
	}{
		{
			name:      "default formatter",
			formatter: nil,
			method:    "GET",
			path:      "/api/users",
			want:      "GET /api/users",
		},
		{
			name: "custom formatter",
			formatter: func(r *http.Request) string {
				return "custom:" + r.Method
			},
			method: "POST",
			path:   "/api/items",
			want:   "custom:POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{SpanNameFormatter: tt.formatter}
			wrapper := cfg.Wrap()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			got := wrapper.GetSpanName(req)
			if got != tt.want {
				t.Errorf("GetSpanName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultSpanNameFormatter(t *testing.T) {
	req := httptest.NewRequest("GET", "/test/path", nil)
	got := DefaultSpanNameFormatter(req)
	want := "GET /test/path"
	if got != want {
		t.Errorf("DefaultSpanNameFormatter() = %q, want %q", got, want)
	}
}

func TestDefaultTracerConfig(t *testing.T) {
	if DefaultConfig.ExcludedPaths == nil {
		t.Error("DefaultTracerConfig.ExcludedPaths should not be nil")
	}
	if len(DefaultConfig.ExcludedPaths) != 0 {
		t.Errorf("DefaultTracerConfig.ExcludedPaths should be empty, got %d", len(DefaultConfig.ExcludedPaths))
	}
	if DefaultConfig.IncludedPaths == nil {
		t.Error("DefaultTracerConfig.IncludedPaths should not be nil")
	}
	if len(DefaultConfig.IncludedPaths) != 0 {
		t.Errorf("DefaultTracerConfig.IncludedPaths should be empty, got %d", len(DefaultConfig.IncludedPaths))
	}
	if DefaultConfig.SpanNameFormatter != nil {
		t.Error("DefaultTracerConfig.SpanNameFormatter should be nil")
	}
}

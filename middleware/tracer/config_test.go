package tracer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
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
			zhtest.AssertEqual(t, tt.want, got)
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
			zhtest.AssertEqual(t, tt.want, got)
		})
	}
}

func TestDefaultSpanNameFormatter(t *testing.T) {
	req := httptest.NewRequest("GET", "/test/path", nil)
	got := DefaultSpanNameFormatter(req)
	want := "GET /test/path"
	zhtest.AssertEqual(t, want, got)
}

func TestDefaultTracerConfig(t *testing.T) {
	zhtest.AssertNotNil(t, DefaultConfig.ExcludedPaths)
	zhtest.AssertEqual(t, 0, len(DefaultConfig.ExcludedPaths))
	zhtest.AssertNotNil(t, DefaultConfig.IncludedPaths)
	zhtest.AssertEqual(t, 0, len(DefaultConfig.IncludedPaths))
	zhtest.AssertNil(t, DefaultConfig.SpanNameFormatter)
}

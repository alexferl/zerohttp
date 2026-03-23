package requestlogger

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/alexferl/zerohttp/log"
)

func TestRequestLoggerConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	if cfg.LogErrors != true {
		t.Errorf("expected default log errors = true, got %t", cfg.LogErrors)
	}
	if len(cfg.Fields) != 13 {
		t.Errorf("expected 13 default fields, got %d", len(cfg.Fields))
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default excluded paths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}

	// Test default field values
	expectedFields := []LogField{
		FieldMethod, FieldURI, FieldPath, FieldHost, FieldProtocol,
		FieldReferer, FieldUserAgent, FieldStatus, FieldDurationNS,
		FieldDurationHuman, FieldRemoteAddr, FieldClientIP, FieldRequestID,
	}
	if !reflect.DeepEqual(cfg.Fields, expectedFields) {
		t.Errorf("expected default fields = %v, got %v", expectedFields, cfg.Fields)
	}
}

func TestRequestLoggerConfig_FieldConstants(t *testing.T) {
	tests := []struct {
		field    LogField
		expected string
	}{
		{FieldMethod, "method"},
		{FieldURI, "uri"},
		{FieldPath, "path"},
		{FieldHost, "host"},
		{FieldProtocol, "protocol"},
		{FieldReferer, "referer"},
		{FieldUserAgent, "user_agent"},
		{FieldStatus, "status"},
		{FieldDurationNS, "duration_ns"},
		{FieldDurationHuman, "duration_human"},
		{FieldRemoteAddr, "remote_addr"},
		{FieldClientIP, "client_ip"},
		{FieldRequestID, "request_id"},
	}

	for _, tt := range tests {
		if string(tt.field) != tt.expected {
			t.Errorf("expected field %s = %s, got %s", tt.expected, tt.expected, string(tt.field))
		}
	}
}

func TestRequestLoggerConfig_StructAssignment(t *testing.T) {
	t.Run("log errors assignment", func(t *testing.T) {
		cfg := Config{
			LogErrors:     false,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: []string{},
		}
		if cfg.LogErrors != false {
			t.Errorf("expected log errors = false, got %t", cfg.LogErrors)
		}
		// Test setting back to true
		cfg.LogErrors = true
		if cfg.LogErrors != true {
			t.Errorf("expected log errors = true, got %t", cfg.LogErrors)
		}
	})

	t.Run("fields assignment", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldPath, FieldStatus, FieldDurationHuman}
		cfg := Config{
			LogErrors:     true,
			Fields:        fields,
			ExcludedPaths: []string{},
		}
		if len(cfg.Fields) != 4 {
			t.Errorf("expected 4 fields, got %d", len(cfg.Fields))
		}
		if !reflect.DeepEqual(cfg.Fields, fields) {
			t.Errorf("expected fields = %v, got %v", fields, cfg.Fields)
		}
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/health", "/metrics", "/ping", "/status"}
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 4 {
			t.Errorf("expected 4 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("multiple fields assignment", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldStatus, FieldDurationHuman}
		excludedPaths := []string{"/health", "/metrics"}
		cfg := Config{
			LogErrors:     false,
			Fields:        fields,
			ExcludedPaths: excludedPaths,
		}

		if cfg.LogErrors != false {
			t.Errorf("expected log errors = false, got %t", cfg.LogErrors)
		}
		if !reflect.DeepEqual(cfg.Fields, fields) {
			t.Error("expected fields to be set correctly")
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Error("expected excluded paths to be set correctly")
		}
	})
}

func TestRequestLoggerConfig_FieldScenarios(t *testing.T) {
	t.Run("minimal fields", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldPath, FieldStatus}
		cfg := Config{
			LogErrors:     true,
			Fields:        fields,
			ExcludedPaths: []string{},
		}
		if len(cfg.Fields) != 3 {
			t.Errorf("expected 3 minimal fields, got %d", len(cfg.Fields))
		}
		if !reflect.DeepEqual(cfg.Fields, fields) {
			t.Errorf("expected fields = %v, got %v", fields, cfg.Fields)
		}
	})

	t.Run("single field variations", func(t *testing.T) {
		allFields := []LogField{
			FieldMethod, FieldURI, FieldPath, FieldHost, FieldProtocol,
			FieldReferer, FieldUserAgent, FieldStatus, FieldDurationNS,
			FieldDurationHuman, FieldRemoteAddr, FieldClientIP, FieldRequestID,
		}

		for _, field := range allFields {
			t.Run(string(field), func(t *testing.T) {
				cfg := Config{
					LogErrors:     true,
					Fields:        []LogField{field},
					ExcludedPaths: []string{},
				}
				if len(cfg.Fields) != 1 {
					t.Errorf("expected 1 field, got %d", len(cfg.Fields))
				}
				if cfg.Fields[0] != field {
					t.Errorf("expected field = %s, got %s", field, cfg.Fields[0])
				}
			})
		}
	})

	t.Run("duration fields", func(t *testing.T) {
		durationFields := []LogField{FieldDurationNS, FieldDurationHuman}
		cfg := Config{
			LogErrors:     true,
			Fields:        durationFields,
			ExcludedPaths: []string{},
		}
		if len(cfg.Fields) != 2 {
			t.Errorf("expected 2 duration fields, got %d", len(cfg.Fields))
		}
		if !reflect.DeepEqual(cfg.Fields, durationFields) {
			t.Errorf("expected duration fields = %v, got %v", durationFields, cfg.Fields)
		}
	})

	t.Run("security fields", func(t *testing.T) {
		securityFields := []LogField{FieldRemoteAddr, FieldClientIP, FieldUserAgent, FieldReferer}
		cfg := Config{
			LogErrors:     true,
			Fields:        securityFields,
			ExcludedPaths: []string{},
		}
		if len(cfg.Fields) != 4 {
			t.Errorf("expected 4 security fields, got %d", len(cfg.Fields))
		}
		if !reflect.DeepEqual(cfg.Fields, securityFields) {
			t.Errorf("expected security fields = %v, got %v", securityFields, cfg.Fields)
		}
	})
}

func TestRequestLoggerConfig_EdgeCases(t *testing.T) {
	t.Run("empty fields", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        []LogField{},
			ExcludedPaths: []string{},
		}
		if cfg.Fields == nil {
			t.Error("expected fields slice to be initialized, not nil")
		}
		if len(cfg.Fields) != 0 {
			t.Errorf("expected empty fields slice, got %d entries", len(cfg.Fields))
		}
	})

	t.Run("nil fields", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        nil,
			ExcludedPaths: []string{},
		}
		if cfg.Fields != nil {
			t.Error("expected fields to remain nil when nil is passed")
		}
	})

	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: []string{},
		}
		if cfg.ExcludedPaths == nil {
			t.Error("expected excluded paths slice to be initialized, not nil")
		}
		if len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected empty excluded paths slice, got %d entries", len(cfg.ExcludedPaths))
		}
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: nil,
		}
		if cfg.ExcludedPaths != nil {
			t.Error("expected excluded paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		excludedPaths := []string{"", "/health", ""}
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 3 {
			t.Errorf("expected 3 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		if cfg.LogErrors != false {
			t.Errorf("expected zero log errors = false, got %t", cfg.LogErrors)
		}
		if cfg.Fields != nil {
			t.Errorf("expected zero fields = nil, got %v", cfg.Fields)
		}
		if cfg.ExcludedPaths != nil {
			t.Errorf("expected zero excluded paths = nil, got %v", cfg.ExcludedPaths)
		}
	})
}

func TestRequestLoggerConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		excludedPaths := []string{
			"/health", "/metrics", "/api/v1/health/*", "/monitoring/*",
			"*.json", "/admin/debug/*", "/internal/status", "/ping",
		}
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		excludedPaths := []string{
			"/api-v1/health", "/metrics_endpoint", "/health-check", "/status.json",
			"/monitoring (internal)", "/path with spaces", "/path/with/unicode-ñ", "/endpoint@service.com",
		}
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != len(excludedPaths) {
			t.Errorf("expected %d excluded paths, got %d", len(excludedPaths), len(cfg.ExcludedPaths))
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, excludedPaths) {
			t.Errorf("expected excluded paths = %v, got %v", excludedPaths, cfg.ExcludedPaths)
		}
	})
}

func TestRequestLoggerConfig_StructCreation(t *testing.T) {
	t.Run("basic struct creation", func(t *testing.T) {
		cfg := Config{
			LogErrors:     false,
			Fields:        []LogField{FieldMethod, FieldStatus},
			ExcludedPaths: []string{"/health", "/metrics"},
		}

		if cfg.LogErrors != false {
			t.Errorf("expected log errors = false, got %t", cfg.LogErrors)
		}
		if !reflect.DeepEqual(cfg.Fields, []LogField{FieldMethod, FieldStatus}) {
			t.Errorf("expected fields = [method status], got %v", cfg.Fields)
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, []string{"/health", "/metrics"}) {
			t.Errorf("expected excluded paths = [/health /metrics], got %v", cfg.ExcludedPaths)
		}
	})

	t.Run("default values copy", func(t *testing.T) {
		cfg := DefaultConfig

		if cfg.LogErrors != DefaultConfig.LogErrors {
			t.Errorf("expected log errors = %t, got %t", DefaultConfig.LogErrors, cfg.LogErrors)
		}
		if !reflect.DeepEqual(cfg.Fields, DefaultConfig.Fields) {
			t.Errorf("expected fields to match default")
		}
		if !reflect.DeepEqual(cfg.ExcludedPaths, DefaultConfig.ExcludedPaths) {
			t.Errorf("expected excluded paths to match default")
		}
	})

	t.Run("logging scenarios", func(t *testing.T) {
		tests := []struct {
			name          string
			logErrors     bool
			fields        []LogField
			excludedPaths []string
		}{
			{"minimal logging", false, []LogField{FieldMethod, FieldPath, FieldStatus}, []string{"/health"}},
			{"verbose logging", true, DefaultConfig.Fields, []string{}},
			{"performance focused", false, []LogField{FieldMethod, FieldStatus, FieldDurationHuman}, []string{"/health", "/metrics", "/ping"}},
			{"security audit", true, []LogField{FieldMethod, FieldPath, FieldRemoteAddr, FieldClientIP, FieldUserAgent, FieldStatus}, []string{}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := Config{
					LogErrors:     tt.logErrors,
					Fields:        tt.fields,
					ExcludedPaths: tt.excludedPaths,
				}

				if cfg.LogErrors != tt.logErrors {
					t.Errorf("expected log errors = %t, got %t", tt.logErrors, cfg.LogErrors)
				}
				if !reflect.DeepEqual(cfg.Fields, tt.fields) {
					t.Errorf("expected fields = %v, got %v", tt.fields, cfg.Fields)
				}
				if !reflect.DeepEqual(cfg.ExcludedPaths, tt.excludedPaths) {
					t.Errorf("expected excluded paths = %v, got %v", tt.excludedPaths, cfg.ExcludedPaths)
				}
			})
		}
	})
}

func TestRequestLoggerConfig_CustomFields(t *testing.T) {
	t.Run("custom fields can be set", func(t *testing.T) {
		customFunc := func(r *http.Request) []log.Field {
			return []log.Field{log.F("api_key", r.Header.Get("X-API-Key"))}
		}
		cfg := Config{
			CustomFields: customFunc,
		}

		if cfg.CustomFields == nil {
			t.Error("expected CustomFields to be set")
		}

		// Verify it works
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "test-key")
		fields := cfg.CustomFields(req)

		if len(fields) != 1 {
			t.Fatalf("expected 1 field, got %d", len(fields))
		}
		if fields[0].Key != "api_key" {
			t.Errorf("expected key 'api_key', got %s", fields[0].Key)
		}
		if fields[0].Value != "test-key" {
			t.Errorf("expected value 'test-key', got %v", fields[0].Value)
		}
	})

	t.Run("custom fields can be nil", func(t *testing.T) {
		cfg := Config{
			CustomFields: nil,
		}

		if cfg.CustomFields != nil {
			t.Error("expected CustomFields to be nil")
		}
	})
}

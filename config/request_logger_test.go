package config

import (
	"reflect"
	"testing"
)

func TestRequestLoggerConfig_DefaultValues(t *testing.T) {
	cfg := DefaultRequestLoggerConfig
	if cfg.LogErrors != true {
		t.Errorf("expected default log errors = true, got %t", cfg.LogErrors)
	}
	if len(cfg.Fields) != 13 {
		t.Errorf("expected 13 default fields, got %d", len(cfg.Fields))
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
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

func TestRequestLoggerOptions(t *testing.T) {
	t.Run("log errors option", func(t *testing.T) {
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerLogErrors(false)(&cfg)
		if cfg.LogErrors != false {
			t.Errorf("expected log errors = false, got %t", cfg.LogErrors)
		}
		// Test setting back to true
		WithRequestLoggerLogErrors(true)(&cfg)
		if cfg.LogErrors != true {
			t.Errorf("expected log errors = true, got %t", cfg.LogErrors)
		}
	})

	t.Run("fields option", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldPath, FieldStatus, FieldDurationHuman}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerFields(fields)(&cfg)
		if len(cfg.Fields) != 4 {
			t.Errorf("expected 4 fields, got %d", len(cfg.Fields))
		}
		if !reflect.DeepEqual(cfg.Fields, fields) {
			t.Errorf("expected fields = %v, got %v", fields, cfg.Fields)
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/health", "/metrics", "/ping", "/status"}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldStatus, FieldDurationHuman}
		exemptPaths := []string{"/health", "/metrics"}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerLogErrors(false)(&cfg)
		WithRequestLoggerFields(fields)(&cfg)
		WithRequestLoggerExemptPaths(exemptPaths)(&cfg)

		if cfg.LogErrors != false {
			t.Errorf("expected log errors = false, got %t", cfg.LogErrors)
		}
		if !reflect.DeepEqual(cfg.Fields, fields) {
			t.Error("expected fields to be set correctly")
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Error("expected exempt paths to be set correctly")
		}
	})
}

func TestRequestLoggerConfig_FieldScenarios(t *testing.T) {
	t.Run("minimal fields", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldPath, FieldStatus}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerFields(fields)(&cfg)
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
				cfg := DefaultRequestLoggerConfig
				WithRequestLoggerFields([]LogField{field})(&cfg)
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
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerFields(durationFields)(&cfg)
		if len(cfg.Fields) != 2 {
			t.Errorf("expected 2 duration fields, got %d", len(cfg.Fields))
		}
		if !reflect.DeepEqual(cfg.Fields, durationFields) {
			t.Errorf("expected duration fields = %v, got %v", durationFields, cfg.Fields)
		}
	})

	t.Run("security fields", func(t *testing.T) {
		securityFields := []LogField{FieldRemoteAddr, FieldClientIP, FieldUserAgent, FieldReferer}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerFields(securityFields)(&cfg)
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
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerFields([]LogField{})(&cfg)
		if cfg.Fields == nil {
			t.Error("expected fields slice to be initialized, not nil")
		}
		if len(cfg.Fields) != 0 {
			t.Errorf("expected empty fields slice, got %d entries", len(cfg.Fields))
		}
	})

	t.Run("nil fields", func(t *testing.T) {
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerFields(nil)(&cfg)
		if cfg.Fields != nil {
			t.Error("expected fields to remain nil when nil is passed")
		}
	})

	t.Run("empty exempt paths", func(t *testing.T) {
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerExemptPaths([]string{})(&cfg)
		if cfg.ExemptPaths == nil {
			t.Error("expected exempt paths slice to be initialized, not nil")
		}
		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %d entries", len(cfg.ExemptPaths))
		}
	})

	t.Run("nil exempt paths", func(t *testing.T) {
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerExemptPaths(nil)(&cfg)
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("empty string paths", func(t *testing.T) {
		exemptPaths := []string{"", "/health", ""}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 3 {
			t.Errorf("expected 3 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := RequestLoggerConfig{} // Zero values
		if cfg.LogErrors != false {
			t.Errorf("expected zero log errors = false, got %t", cfg.LogErrors)
		}
		if cfg.Fields != nil {
			t.Errorf("expected zero fields = nil, got %v", cfg.Fields)
		}
		if cfg.ExemptPaths != nil {
			t.Errorf("expected zero exempt paths = nil, got %v", cfg.ExemptPaths)
		}
	})
}

func TestRequestLoggerConfig_PathPatterns(t *testing.T) {
	t.Run("pattern paths", func(t *testing.T) {
		exemptPaths := []string{
			"/health", "/metrics", "/api/v1/health/*", "/monitoring/*",
			"*.json", "/admin/debug/*", "/internal/status", "/ping",
		}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})

	t.Run("special character paths", func(t *testing.T) {
		exemptPaths := []string{
			"/api-v1/health", "/metrics_endpoint", "/health-check", "/status.json",
			"/monitoring (internal)", "/path with spaces", "/path/with/unicode-ñ", "/endpoint@service.com",
		}
		cfg := DefaultRequestLoggerConfig
		WithRequestLoggerExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != len(exemptPaths) {
			t.Errorf("expected %d exempt paths, got %d", len(exemptPaths), len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestRequestLoggerConfigToOptions(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		cfg := RequestLoggerConfig{
			LogErrors:   false,
			Fields:      []LogField{FieldMethod, FieldStatus},
			ExemptPaths: []string{"/health", "/metrics"},
		}

		options := requestLoggerConfigToOptions(cfg)
		if len(options) != 3 {
			t.Errorf("expected 3 options, got %d", len(options))
		}

		newCfg := DefaultRequestLoggerConfig
		for _, option := range options {
			option(&newCfg)
		}

		if newCfg.LogErrors != false {
			t.Errorf("expected converted log errors = false, got %t", newCfg.LogErrors)
		}
		if !reflect.DeepEqual(newCfg.Fields, []LogField{FieldMethod, FieldStatus}) {
			t.Errorf("expected converted fields = [method status], got %v", newCfg.Fields)
		}
		if !reflect.DeepEqual(newCfg.ExemptPaths, []string{"/health", "/metrics"}) {
			t.Errorf("expected converted exempt paths = [/health /metrics], got %v", newCfg.ExemptPaths)
		}
	})

	t.Run("default values conversion", func(t *testing.T) {
		cfg := DefaultRequestLoggerConfig
		options := requestLoggerConfigToOptions(cfg)
		if len(options) != 3 {
			t.Errorf("expected 3 options for default config, got %d", len(options))
		}

		newCfg := RequestLoggerConfig{} // Start with zero values
		for _, option := range options {
			option(&newCfg)
		}

		if newCfg.LogErrors != DefaultRequestLoggerConfig.LogErrors {
			t.Errorf("expected converted log errors = %t, got %t", DefaultRequestLoggerConfig.LogErrors, newCfg.LogErrors)
		}
		if !reflect.DeepEqual(newCfg.Fields, DefaultRequestLoggerConfig.Fields) {
			t.Errorf("expected converted fields to match default")
		}
		if !reflect.DeepEqual(newCfg.ExemptPaths, DefaultRequestLoggerConfig.ExemptPaths) {
			t.Errorf("expected converted exempt paths to match default")
		}
	})

	t.Run("logging scenarios", func(t *testing.T) {
		tests := []struct {
			name        string
			logErrors   bool
			fields      []LogField
			exemptPaths []string
		}{
			{"minimal logging", false, []LogField{FieldMethod, FieldPath, FieldStatus}, []string{"/health"}},
			{"verbose logging", true, DefaultRequestLoggerConfig.Fields, []string{}},
			{"performance focused", false, []LogField{FieldMethod, FieldStatus, FieldDurationHuman}, []string{"/health", "/metrics", "/ping"}},
			{"security audit", true, []LogField{FieldMethod, FieldPath, FieldRemoteAddr, FieldClientIP, FieldUserAgent, FieldStatus}, []string{}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := RequestLoggerConfig{
					LogErrors:   tt.logErrors,
					Fields:      tt.fields,
					ExemptPaths: tt.exemptPaths,
				}

				options := requestLoggerConfigToOptions(cfg)
				newCfg := RequestLoggerConfig{}
				for _, option := range options {
					option(&newCfg)
				}

				if newCfg.LogErrors != tt.logErrors {
					t.Errorf("expected log errors = %t, got %t", tt.logErrors, newCfg.LogErrors)
				}
				if !reflect.DeepEqual(newCfg.Fields, tt.fields) {
					t.Errorf("expected fields = %v, got %v", tt.fields, newCfg.Fields)
				}
				if !reflect.DeepEqual(newCfg.ExemptPaths, tt.exemptPaths) {
					t.Errorf("expected exempt paths = %v, got %v", tt.exemptPaths, newCfg.ExemptPaths)
				}
			})
		}
	})

	t.Run("options equivalence", func(t *testing.T) {
		originalCfg := RequestLoggerConfig{
			LogErrors:   false,
			Fields:      []LogField{FieldMethod, FieldPath, FieldStatus, FieldDurationHuman},
			ExemptPaths: []string{"/health", "/ping"},
		}

		// Method 1: Apply options individually
		cfg1 := DefaultRequestLoggerConfig
		WithRequestLoggerLogErrors(originalCfg.LogErrors)(&cfg1)
		WithRequestLoggerFields(originalCfg.Fields)(&cfg1)
		WithRequestLoggerExemptPaths(originalCfg.ExemptPaths)(&cfg1)

		// Method 2: Apply via requestLoggerConfigToOptions
		cfg2 := DefaultRequestLoggerConfig
		options := requestLoggerConfigToOptions(originalCfg)
		for _, option := range options {
			option(&cfg2)
		}

		// Both should be identical
		if !reflect.DeepEqual(cfg1, cfg2) {
			t.Errorf("configurations should be identical: cfg1=%+v, cfg2=%+v", cfg1, cfg2)
		}
	})
}

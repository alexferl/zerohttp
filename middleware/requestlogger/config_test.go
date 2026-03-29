package requestlogger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequestLoggerConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertTrue(t, cfg.LogErrors)
	zhtest.AssertEqual(t, 13, len(cfg.Fields))
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))

	// Test default field values
	expectedFields := []LogField{
		FieldMethod, FieldURI, FieldPath, FieldHost, FieldProtocol,
		FieldReferer, FieldUserAgent, FieldStatus, FieldDurationNS,
		FieldDurationHuman, FieldRemoteAddr, FieldClientIP, FieldRequestID,
	}
	zhtest.AssertEqual(t, expectedFields, cfg.Fields)
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
		zhtest.AssertEqual(t, tt.expected, string(tt.field))
	}
}

func TestRequestLoggerConfig_StructAssignment(t *testing.T) {
	t.Run("log errors assignment", func(t *testing.T) {
		cfg := Config{
			LogErrors:     false,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: []string{},
		}
		zhtest.AssertFalse(t, cfg.LogErrors)
		// Test setting back to true
		cfg.LogErrors = true
		zhtest.AssertTrue(t, cfg.LogErrors)
	})

	t.Run("fields assignment", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldPath, FieldStatus, FieldDurationHuman}
		cfg := Config{
			LogErrors:     true,
			Fields:        fields,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, 4, len(cfg.Fields))
		zhtest.AssertEqual(t, fields, cfg.Fields)
	})

	t.Run("excluded paths assignment", func(t *testing.T) {
		excludedPaths := []string{"/health", "/metrics", "/ping", "/status"}
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("multiple fields assignment", func(t *testing.T) {
		fields := []LogField{FieldMethod, FieldStatus, FieldDurationHuman}
		excludedPaths := []string{"/health", "/metrics"}
		cfg := Config{
			LogErrors:     false,
			Fields:        fields,
			ExcludedPaths: excludedPaths,
		}

		zhtest.AssertFalse(t, cfg.LogErrors)
		zhtest.AssertEqual(t, fields, cfg.Fields)
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
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
		zhtest.AssertEqual(t, 3, len(cfg.Fields))
		zhtest.AssertEqual(t, fields, cfg.Fields)
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
				zhtest.AssertEqual(t, 1, len(cfg.Fields))
				zhtest.AssertEqual(t, field, cfg.Fields[0])
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
		zhtest.AssertEqual(t, 2, len(cfg.Fields))
		zhtest.AssertEqual(t, durationFields, cfg.Fields)
	})

	t.Run("security fields", func(t *testing.T) {
		securityFields := []LogField{FieldRemoteAddr, FieldClientIP, FieldUserAgent, FieldReferer}
		cfg := Config{
			LogErrors:     true,
			Fields:        securityFields,
			ExcludedPaths: []string{},
		}
		zhtest.AssertEqual(t, 4, len(cfg.Fields))
		zhtest.AssertEqual(t, securityFields, cfg.Fields)
	})
}

func TestRequestLoggerConfig_EdgeCases(t *testing.T) {
	t.Run("empty fields", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        []LogField{},
			ExcludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.Fields)
		zhtest.AssertEqual(t, 0, len(cfg.Fields))
	})

	t.Run("nil fields", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        nil,
			ExcludedPaths: []string{},
		}
		zhtest.AssertNil(t, cfg.Fields)
	})

	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.ExcludedPaths)
	})

	t.Run("empty string paths", func(t *testing.T) {
		excludedPaths := []string{"", "/health", ""}
		cfg := Config{
			LogErrors:     true,
			Fields:        DefaultConfig.Fields,
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 3, len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertFalse(t, cfg.LogErrors)
		zhtest.AssertNil(t, cfg.Fields)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
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
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
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
		zhtest.AssertEqual(t, len(excludedPaths), len(cfg.ExcludedPaths))
		zhtest.AssertEqual(t, excludedPaths, cfg.ExcludedPaths)
	})
}

func TestRequestLoggerConfig_StructCreation(t *testing.T) {
	t.Run("basic struct creation", func(t *testing.T) {
		cfg := Config{
			LogErrors:     false,
			Fields:        []LogField{FieldMethod, FieldStatus},
			ExcludedPaths: []string{"/health", "/metrics"},
		}

		zhtest.AssertFalse(t, cfg.LogErrors)
		zhtest.AssertEqual(t, []LogField{FieldMethod, FieldStatus}, cfg.Fields)
		zhtest.AssertEqual(t, []string{"/health", "/metrics"}, cfg.ExcludedPaths)
	})

	t.Run("default values copy", func(t *testing.T) {
		cfg := DefaultConfig

		zhtest.AssertEqual(t, DefaultConfig.LogErrors, cfg.LogErrors)
		zhtest.AssertEqual(t, DefaultConfig.Fields, cfg.Fields)
		zhtest.AssertEqual(t, DefaultConfig.ExcludedPaths, cfg.ExcludedPaths)
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

				zhtest.AssertEqual(t, tt.logErrors, cfg.LogErrors)
				zhtest.AssertEqual(t, tt.fields, cfg.Fields)
				zhtest.AssertEqual(t, tt.excludedPaths, cfg.ExcludedPaths)
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

		zhtest.AssertNotNil(t, cfg.CustomFields)

		// Verify it works
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "test-key")
		fields := cfg.CustomFields(req)

		zhtest.AssertEqual(t, 1, len(fields))
		zhtest.AssertEqual(t, "api_key", fields[0].Key)
		zhtest.AssertEqual(t, "test-key", fields[0].Value)
	})

	t.Run("custom fields can be nil", func(t *testing.T) {
		cfg := Config{
			CustomFields: nil,
		}

		zhtest.AssertNil(t, cfg.CustomFields)
	})
}

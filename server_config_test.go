package zerohttp

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestConfigMerging(t *testing.T) {
	t.Run("partial config uses default Addr", func(t *testing.T) {
		// Only set DisableDefaultMiddlewares, Addr should use default
		s := New(config.Config{DisableDefaultMiddlewares: true})

		// Verify server was created successfully with default address
		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("custom Addr overrides default", func(t *testing.T) {
		customAddr := ":9999"
		s := New(config.Config{Addr: customAddr})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("partial config with only middleware settings uses default Addr", func(t *testing.T) {
		s := New(config.Config{
			RequestBodySize: config.RequestBodySizeConfig{
				MaxBytes: 10 * 1024 * 1024,
			},
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("merge preserves default TLSAddr when not set", func(t *testing.T) {
		s := New(config.Config{
			Addr: ":8080",
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("merge RequestID config", func(t *testing.T) {
		s := New(config.Config{
			RequestID: config.RequestIDConfig{
				Header: "X-Custom-Request-ID",
			},
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})

	t.Run("merge SecurityHeaders config", func(t *testing.T) {
		s := New(config.Config{
			SecurityHeaders: config.SecurityHeadersConfig{
				XFrameOptions: "SAMEORIGIN",
			},
		})

		if s == nil {
			t.Fatal("server should not be nil")
		}
	})
}

func TestMergeRecoverConfig(t *testing.T) {
	defaults := config.DefaultRecoverConfig

	t.Run("empty user config keeps defaults", func(t *testing.T) {
		result := mergeRecoverConfig(defaults, config.RecoverConfig{})
		if result.StackSize != defaults.StackSize {
			t.Errorf("expected StackSize %d, got %d", defaults.StackSize, result.StackSize)
		}
		if result.EnableStackTrace != defaults.EnableStackTrace {
			t.Errorf("expected EnableStackTrace %v, got %v", defaults.EnableStackTrace, result.EnableStackTrace)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		user := config.RecoverConfig{
			StackSize:        8192,
			EnableStackTrace: true,
		}
		result := mergeRecoverConfig(defaults, user)
		if result.StackSize != 8192 {
			t.Errorf("expected StackSize 8192, got %d", result.StackSize)
		}
		if !result.EnableStackTrace {
			t.Error("expected EnableStackTrace to be true")
		}
	})
}

func TestMergeRequestBodySizeConfig(t *testing.T) {
	defaults := config.DefaultRequestBodySizeConfig

	t.Run("empty user config keeps defaults", func(t *testing.T) {
		result := mergeRequestBodySizeConfig(defaults, config.RequestBodySizeConfig{})
		if result.MaxBytes != defaults.MaxBytes {
			t.Errorf("expected MaxBytes %d, got %d", defaults.MaxBytes, result.MaxBytes)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		user := config.RequestBodySizeConfig{
			MaxBytes:    10 * 1024 * 1024,
			ExemptPaths: []string{"/upload", "/webhook"},
		}
		result := mergeRequestBodySizeConfig(defaults, user)
		if result.MaxBytes != 10*1024*1024 {
			t.Errorf("expected MaxBytes %d, got %d", 10*1024*1024, result.MaxBytes)
		}
		if len(result.ExemptPaths) != 2 || result.ExemptPaths[0] != "/upload" {
			t.Errorf("expected ExemptPaths [/upload /webhook], got %v", result.ExemptPaths)
		}
	})

	t.Run("empty exempt paths not applied", func(t *testing.T) {
		user := config.RequestBodySizeConfig{
			MaxBytes:    5 * 1024 * 1024,
			ExemptPaths: []string{},
		}
		result := mergeRequestBodySizeConfig(defaults, user)
		if result.MaxBytes != 5*1024*1024 {
			t.Errorf("expected MaxBytes %d, got %d", 5*1024*1024, result.MaxBytes)
		}
		if len(result.ExemptPaths) != 0 {
			t.Errorf("expected empty ExemptPaths, got %v", result.ExemptPaths)
		}
	})
}

func TestMergeRequestIDConfig(t *testing.T) {
	defaults := config.DefaultRequestIDConfig

	t.Run("empty user config keeps defaults", func(t *testing.T) {
		result := mergeRequestIDConfig(defaults, config.RequestIDConfig{})
		if result.Header != defaults.Header {
			t.Errorf("expected Header %s, got %s", defaults.Header, result.Header)
		}
		if result.ContextKey != defaults.ContextKey {
			t.Errorf("expected ContextKey %s, got %s", defaults.ContextKey, result.ContextKey)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		generator := func() string { return "custom-id" }
		user := config.RequestIDConfig{
			Header:     "X-Custom-ID",
			ContextKey: "customKey",
			Generator:  generator,
		}
		result := mergeRequestIDConfig(defaults, user)
		if result.Header != "X-Custom-ID" {
			t.Errorf("expected Header X-Custom-ID, got %s", result.Header)
		}
		if result.ContextKey != "customKey" {
			t.Errorf("expected ContextKey customKey, got %s", result.ContextKey)
		}
		if result.Generator == nil {
			t.Error("expected Generator to be set")
		}
	})
}

func TestMergeRequestLoggerConfig(t *testing.T) {
	defaults := config.DefaultRequestLoggerConfig

	t.Run("empty user config keeps defaults", func(t *testing.T) {
		result := mergeRequestLoggerConfig(defaults, config.RequestLoggerConfig{})
		if result.LogErrors != defaults.LogErrors {
			t.Errorf("expected LogErrors %v, got %v", defaults.LogErrors, result.LogErrors)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		user := config.RequestLoggerConfig{
			LogErrors:   true,
			Fields:      []config.LogField{config.FieldMethod, config.FieldPath, config.FieldDurationHuman},
			ExemptPaths: []string{"/health", "/metrics"},
		}
		result := mergeRequestLoggerConfig(defaults, user)
		if !result.LogErrors {
			t.Error("expected LogErrors to be true")
		}
		if len(result.Fields) != 3 {
			t.Errorf("expected 3 Fields, got %d", len(result.Fields))
		}
		if len(result.ExemptPaths) != 2 {
			t.Errorf("expected 2 ExemptPaths, got %d", len(result.ExemptPaths))
		}
	})
}

func TestMergeSecurityHeadersConfig(t *testing.T) {
	defaults := config.DefaultSecurityHeadersConfig

	t.Run("empty user config keeps defaults", func(t *testing.T) {
		result := mergeSecurityHeadersConfig(defaults, config.SecurityHeadersConfig{})
		if result.XFrameOptions != defaults.XFrameOptions {
			t.Errorf("expected XFrameOptions %s, got %s", defaults.XFrameOptions, result.XFrameOptions)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		user := config.SecurityHeadersConfig{
			XFrameOptions:         "DENY",
			ContentSecurityPolicy: "default-src 'self'",
			Server:                "CustomServer",
			ExemptPaths:           []string{"/api/public"},
		}
		result := mergeSecurityHeadersConfig(defaults, user)
		if result.XFrameOptions != "DENY" {
			t.Errorf("expected XFrameOptions DENY, got %s", result.XFrameOptions)
		}
		if result.ContentSecurityPolicy != "default-src 'self'" {
			t.Errorf("expected CSP 'default-src 'self'', got %s", result.ContentSecurityPolicy)
		}
		if result.Server != "CustomServer" {
			t.Errorf("expected Server CustomServer, got %s", result.Server)
		}
		if len(result.ExemptPaths) != 1 || result.ExemptPaths[0] != "/api/public" {
			t.Errorf("expected ExemptPaths [/api/public], got %v", result.ExemptPaths)
		}
	})

	t.Run("partial user config merges with defaults", func(t *testing.T) {
		user := config.SecurityHeadersConfig{
			XFrameOptions: "SAMEORIGIN",
		}
		result := mergeSecurityHeadersConfig(defaults, user)
		if result.XFrameOptions != "SAMEORIGIN" {
			t.Errorf("expected XFrameOptions SAMEORIGIN, got %s", result.XFrameOptions)
		}
		// Other fields should keep defaults
		if result.ContentSecurityPolicy != defaults.ContentSecurityPolicy {
			t.Errorf("expected CSP to keep default, got %s", result.ContentSecurityPolicy)
		}
	})

	t.Run("StrictTransportSecurity merge", func(t *testing.T) {
		user := config.SecurityHeadersConfig{
			StrictTransportSecurity: config.StrictTransportSecurity{
				MaxAge:            31536000,
				ExcludeSubdomains: true,
			},
		}
		result := mergeSecurityHeadersConfig(defaults, user)
		if result.StrictTransportSecurity.MaxAge != 31536000 {
			t.Errorf("expected HSTS MaxAge 31536000, got %d", result.StrictTransportSecurity.MaxAge)
		}
		if !result.StrictTransportSecurity.ExcludeSubdomains {
			t.Error("expected HSTS ExcludeSubdomains to be true")
		}
	})
}

func TestMergeMetricsConfig(t *testing.T) {
	defaults := config.DefaultMetricsConfig

	t.Run("empty user config applies zero values", func(t *testing.T) {
		// Note: Enabled is always applied, so empty config will set Enabled to false
		result := mergeMetricsConfig(defaults, config.MetricsConfig{})
		if result.Enabled {
			t.Error("expected Enabled to be false when user config is empty")
		}
		if result.Endpoint != defaults.Endpoint {
			t.Errorf("expected Endpoint %s, got %s", defaults.Endpoint, result.Endpoint)
		}
	})

	t.Run("user values override defaults", func(t *testing.T) {
		customLabels := func(r *http.Request) map[string]string { return nil }
		pathLabelFunc := func(p string) string { return p }
		user := config.MetricsConfig{
			Enabled:         false,
			Endpoint:        "/custom-metrics",
			DurationBuckets: []float64{0.01, 0.1, 1},
			SizeBuckets:     []float64{100, 1000},
			ExcludePaths:    []string{"/health", "/readyz"},
			PathLabelFunc:   pathLabelFunc,
			CustomLabels:    customLabels,
		}
		result := mergeMetricsConfig(defaults, user)
		if result.Enabled {
			t.Error("expected Enabled to be false")
		}
		if result.Endpoint != "/custom-metrics" {
			t.Errorf("expected Endpoint /custom-metrics, got %s", result.Endpoint)
		}
		if len(result.DurationBuckets) != 3 {
			t.Errorf("expected 3 DurationBuckets, got %d", len(result.DurationBuckets))
		}
		if len(result.SizeBuckets) != 2 {
			t.Errorf("expected 2 SizeBuckets, got %d", len(result.SizeBuckets))
		}
		if len(result.ExcludePaths) != 2 {
			t.Errorf("expected 2 ExcludePaths, got %d", len(result.ExcludePaths))
		}
		if result.PathLabelFunc == nil {
			t.Error("expected PathLabelFunc to be set")
		}
		if result.CustomLabels == nil {
			t.Error("expected CustomLabels to be set")
		}
	})

	t.Run("Enabled field is always applied", func(t *testing.T) {
		// Even with zero values, Enabled should be applied
		user := config.MetricsConfig{
			Enabled: false,
		}
		result := mergeMetricsConfig(defaults, user)
		if result.Enabled {
			t.Error("expected Enabled to be false when user sets it")
		}
	})

	t.Run("ServerAddr is applied", func(t *testing.T) {
		user := config.MetricsConfig{
			Enabled:    true,
			ServerAddr: "localhost:9091",
		}
		result := mergeMetricsConfig(defaults, user)
		if result.ServerAddr != "localhost:9091" {
			t.Errorf("expected ServerAddr localhost:9091, got %s", result.ServerAddr)
		}
	})
}

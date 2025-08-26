package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestSecurityHeadersConfig_DefaultValues(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig
	expectedCSP := "default-src 'none'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'self'; frame-ancestors 'self'; form-action 'self';"
	if cfg.ContentSecurityPolicy != expectedCSP {
		t.Errorf("expected default CSP to be set")
	}
	if cfg.ContentSecurityPolicyReportOnly != false {
		t.Errorf("expected default CSP report only = false, got %t", cfg.ContentSecurityPolicyReportOnly)
	}
	if cfg.CrossOriginEmbedderPolicy != "require-corp" {
		t.Errorf("expected default COEP = 'require-corp', got %s", cfg.CrossOriginEmbedderPolicy)
	}
	if cfg.CrossOriginOpenerPolicy != "same-origin" {
		t.Errorf("expected default COOP = 'same-origin', got %s", cfg.CrossOriginOpenerPolicy)
	}
	if cfg.CrossOriginResourcePolicy != "same-origin" {
		t.Errorf("expected default CORP = 'same-origin', got %s", cfg.CrossOriginResourcePolicy)
	}
	if cfg.PermissionsPolicy == "" {
		t.Error("expected default permissions policy to be set")
	}
	if cfg.ReferrerPolicy != "no-referrer" {
		t.Errorf("expected default referrer policy = 'no-referrer', got %s", cfg.ReferrerPolicy)
	}
	if cfg.Server != "" {
		t.Errorf("expected default server = '', got %s", cfg.Server)
	}
	if cfg.XContentTypeOptions != "nosniff" {
		t.Errorf("expected default X-Content-Type-Options = 'nosniff', got %s", cfg.XContentTypeOptions)
	}
	if cfg.XFrameOptions != "DENY" {
		t.Errorf("expected default X-Frame-Options = 'DENY', got %s", cfg.XFrameOptions)
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}

	// Test default HSTS values
	if cfg.StrictTransportSecurity.MaxAge != 0 {
		t.Errorf("expected default HSTS max age = 0, got %d", cfg.StrictTransportSecurity.MaxAge)
	}
	if cfg.StrictTransportSecurity.ExcludeSubdomains != false {
		t.Errorf("expected default HSTS exclude subdomains = false, got %t", cfg.StrictTransportSecurity.ExcludeSubdomains)
	}
	if cfg.StrictTransportSecurity.PreloadEnabled != false {
		t.Errorf("expected default HSTS preload = false, got %t", cfg.StrictTransportSecurity.PreloadEnabled)
	}

	// Test permissions policy
	expectedPolicy := strings.Join(permissionPolicyFeatures, ", ")
	if cfg.PermissionsPolicy != expectedPolicy {
		t.Errorf("expected default permissions policy to match joined features")
	}
}

func TestStrictTransportSecurity_DefaultValues(t *testing.T) {
	hsts := DefaultStrictTransportSecurity
	if hsts.MaxAge != 0 {
		t.Errorf("expected default HSTS max age = 0, got %d", hsts.MaxAge)
	}
	if hsts.ExcludeSubdomains != false {
		t.Errorf("expected default HSTS exclude subdomains = false, got %t", hsts.ExcludeSubdomains)
	}
	if hsts.PreloadEnabled != false {
		t.Errorf("expected default HSTS preload = false, got %t", hsts.PreloadEnabled)
	}
}

func TestPermissionPolicyFeatures(t *testing.T) {
	if len(permissionPolicyFeatures) == 0 {
		t.Error("expected permission policy features to not be empty")
	}

	// Test feature format
	for _, feature := range permissionPolicyFeatures {
		if !strings.HasSuffix(feature, "=()") {
			t.Errorf("expected feature %s to end with '=()'", feature)
		}
	}

	// Test specific expected features
	expectedFeatures := []string{"camera=()", "microphone=()", "geolocation=()", "fullscreen=()", "payment=()"}
	featureMap := make(map[string]bool)
	for _, feature := range permissionPolicyFeatures {
		featureMap[feature] = true
	}
	for _, expected := range expectedFeatures {
		if !featureMap[expected] {
			t.Errorf("expected feature %s to be in permission policy features", expected)
		}
	}
}

func TestSecurityHeadersOptions(t *testing.T) {
	t.Run("basic header options", func(t *testing.T) {
		cfg := DefaultSecurityHeadersConfig

		customCSP := "default-src 'self'; script-src 'self' 'unsafe-inline'"
		WithSecurityHeadersCSP(customCSP)(&cfg)
		if cfg.ContentSecurityPolicy != customCSP {
			t.Errorf("expected CSP = %s, got %s", customCSP, cfg.ContentSecurityPolicy)
		}

		WithSecurityHeadersCSPReportOnly(true)(&cfg)
		if cfg.ContentSecurityPolicyReportOnly != true {
			t.Errorf("expected CSP report only = true, got %t", cfg.ContentSecurityPolicyReportOnly)
		}

		WithSecurityHeadersCrossOriginEmbedderPolicy("unsafe-none")(&cfg)
		if cfg.CrossOriginEmbedderPolicy != "unsafe-none" {
			t.Errorf("expected COEP = 'unsafe-none', got %s", cfg.CrossOriginEmbedderPolicy)
		}

		WithSecurityHeadersCrossOriginOpenerPolicy("unsafe-none")(&cfg)
		if cfg.CrossOriginOpenerPolicy != "unsafe-none" {
			t.Errorf("expected COOP = 'unsafe-none', got %s", cfg.CrossOriginOpenerPolicy)
		}

		WithSecurityHeadersCrossOriginResourcePolicy("cross-origin")(&cfg)
		if cfg.CrossOriginResourcePolicy != "cross-origin" {
			t.Errorf("expected CORP = 'cross-origin', got %s", cfg.CrossOriginResourcePolicy)
		}
	})

	t.Run("policy and server options", func(t *testing.T) {
		cfg := DefaultSecurityHeadersConfig

		customPolicy := "camera=(), microphone=(), geolocation=()"
		WithSecurityHeadersPermissionsPolicy(customPolicy)(&cfg)
		if cfg.PermissionsPolicy != customPolicy {
			t.Errorf("expected permissions policy = %s, got %s", customPolicy, cfg.PermissionsPolicy)
		}

		WithSecurityHeadersReferrerPolicy("strict-origin")(&cfg)
		if cfg.ReferrerPolicy != "strict-origin" {
			t.Errorf("expected referrer policy = 'strict-origin', got %s", cfg.ReferrerPolicy)
		}

		WithSecurityHeadersServer("nginx/1.18.0")(&cfg)
		if cfg.Server != "nginx/1.18.0" {
			t.Errorf("expected server = 'nginx/1.18.0', got %s", cfg.Server)
		}

		WithSecurityHeadersXContentTypeOptions("")(&cfg) // Disable
		if cfg.XContentTypeOptions != "" {
			t.Errorf("expected X-Content-Type-Options = '', got %s", cfg.XContentTypeOptions)
		}

		WithSecurityHeadersXFrameOptions("SAMEORIGIN")(&cfg)
		if cfg.XFrameOptions != "SAMEORIGIN" {
			t.Errorf("expected X-Frame-Options = 'SAMEORIGIN', got %s", cfg.XFrameOptions)
		}
	})

	t.Run("HSTS options", func(t *testing.T) {
		cfg := DefaultSecurityHeadersConfig
		WithSecurityHeadersHSTS(
			WithHSTSMaxAge(31536000),
			WithHSTSExcludeSubdomains(true),
			WithHSTSPreload(true),
		)(&cfg)

		if cfg.StrictTransportSecurity.MaxAge != 31536000 {
			t.Errorf("expected HSTS max age = 31536000, got %d", cfg.StrictTransportSecurity.MaxAge)
		}
		if cfg.StrictTransportSecurity.ExcludeSubdomains != true {
			t.Errorf("expected HSTS exclude subdomains = true, got %t", cfg.StrictTransportSecurity.ExcludeSubdomains)
		}
		if cfg.StrictTransportSecurity.PreloadEnabled != true {
			t.Errorf("expected HSTS preload = true, got %t", cfg.StrictTransportSecurity.PreloadEnabled)
		}

		// Test individual HSTS options
		hsts := DefaultStrictTransportSecurity
		WithHSTSMaxAge(31536000)(&hsts)
		WithHSTSExcludeSubdomains(true)(&hsts)
		WithHSTSPreload(true)(&hsts)
		if hsts.MaxAge != 31536000 || hsts.ExcludeSubdomains != true || hsts.PreloadEnabled != true {
			t.Error("expected individual HSTS options to work correctly")
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/api/webhook", "/health", "/metrics"}
		cfg := DefaultSecurityHeadersConfig
		WithSecurityHeadersExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 3 {
			t.Errorf("expected 3 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
			t.Errorf("expected exempt paths = %v, got %v", exemptPaths, cfg.ExemptPaths)
		}
	})
}

func TestSecurityHeadersConfig_MultipleOptions(t *testing.T) {
	exemptPaths := []string{"/public", "/api/webhook"}
	cfg := DefaultSecurityHeadersConfig
	WithSecurityHeadersCSP("default-src 'self'")(&cfg)
	WithSecurityHeadersCSPReportOnly(true)(&cfg)
	WithSecurityHeadersCrossOriginEmbedderPolicy("unsafe-none")(&cfg)
	WithSecurityHeadersReferrerPolicy("strict-origin")(&cfg)
	WithSecurityHeadersServer("MyServer/1.0")(&cfg)
	WithSecurityHeadersXFrameOptions("SAMEORIGIN")(&cfg)
	WithSecurityHeadersExemptPaths(exemptPaths)(&cfg)

	if cfg.ContentSecurityPolicy != "default-src 'self'" {
		t.Error("expected CSP to be set correctly")
	}
	if cfg.ContentSecurityPolicyReportOnly != true {
		t.Error("expected CSP report only to be true")
	}
	if cfg.CrossOriginEmbedderPolicy != "unsafe-none" {
		t.Error("expected COEP to be set correctly")
	}
	if cfg.ReferrerPolicy != "strict-origin" {
		t.Error("expected referrer policy to be set correctly")
	}
	if cfg.Server != "MyServer/1.0" {
		t.Error("expected server to be set correctly")
	}
	if cfg.XFrameOptions != "SAMEORIGIN" {
		t.Error("expected X-Frame-Options to be set correctly")
	}
	if !reflect.DeepEqual(cfg.ExemptPaths, exemptPaths) {
		t.Error("expected exempt paths to be set correctly")
	}
}

func TestSecurityHeadersConfig_PolicyVariations(t *testing.T) {
	t.Run("CSP variations", func(t *testing.T) {
		tests := []struct {
			name string
			csp  string
		}{
			{"strict", "default-src 'none'"},
			{"self only", "default-src 'self'"},
			{"with unsafe inline", "default-src 'self'; script-src 'self' 'unsafe-inline'"},
			{"with CDN", "default-src 'self'; script-src 'self' https://cdn.example.com"},
			{"empty", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultSecurityHeadersConfig
				WithSecurityHeadersCSP(tt.csp)(&cfg)
				if cfg.ContentSecurityPolicy != tt.csp {
					t.Errorf("expected CSP = %s, got %s", tt.csp, cfg.ContentSecurityPolicy)
				}
			})
		}
	})

	t.Run("cross-origin policies", func(t *testing.T) {
		tests := []struct {
			name string
			coep string
			coop string
			corp string
		}{
			{"strict", "require-corp", "same-origin", "same-origin"},
			{"relaxed", "unsafe-none", "unsafe-none", "cross-origin"},
			{"mixed", "require-corp", "unsafe-none", "same-site"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultSecurityHeadersConfig
				WithSecurityHeadersCrossOriginEmbedderPolicy(tt.coep)(&cfg)
				WithSecurityHeadersCrossOriginOpenerPolicy(tt.coop)(&cfg)
				WithSecurityHeadersCrossOriginResourcePolicy(tt.corp)(&cfg)

				if cfg.CrossOriginEmbedderPolicy != tt.coep {
					t.Errorf("expected COEP = %s, got %s", tt.coep, cfg.CrossOriginEmbedderPolicy)
				}
				if cfg.CrossOriginOpenerPolicy != tt.coop {
					t.Errorf("expected COOP = %s, got %s", tt.coop, cfg.CrossOriginOpenerPolicy)
				}
				if cfg.CrossOriginResourcePolicy != tt.corp {
					t.Errorf("expected CORP = %s, got %s", tt.corp, cfg.CrossOriginResourcePolicy)
				}
			})
		}
	})

	t.Run("referrer policy values", func(t *testing.T) {
		policies := []string{
			"no-referrer", "no-referrer-when-downgrade", "origin", "origin-when-cross-origin",
			"same-origin", "strict-origin", "strict-origin-when-cross-origin", "unsafe-url",
		}

		for _, policy := range policies {
			t.Run(policy, func(t *testing.T) {
				cfg := DefaultSecurityHeadersConfig
				WithSecurityHeadersReferrerPolicy(policy)(&cfg)
				if cfg.ReferrerPolicy != policy {
					t.Errorf("expected referrer policy = %s, got %s", policy, cfg.ReferrerPolicy)
				}
			})
		}
	})

	t.Run("X-Frame-Options values", func(t *testing.T) {
		options := []string{"DENY", "SAMEORIGIN", "ALLOW-FROM https://example.com", ""}

		for _, option := range options {
			t.Run(option, func(t *testing.T) {
				cfg := DefaultSecurityHeadersConfig
				WithSecurityHeadersXFrameOptions(option)(&cfg)
				if cfg.XFrameOptions != option {
					t.Errorf("expected X-Frame-Options = %s, got %s", option, cfg.XFrameOptions)
				}
			})
		}
	})
}

func TestSecurityHeadersConfig_EdgeCases(t *testing.T) {
	t.Run("empty exempt paths", func(t *testing.T) {
		cfg := DefaultSecurityHeadersConfig
		WithSecurityHeadersExemptPaths([]string{})(&cfg)
		if cfg.ExemptPaths == nil {
			t.Error("expected exempt paths slice to be initialized, not nil")
		}
		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %d entries", len(cfg.ExemptPaths))
		}
	})

	t.Run("nil exempt paths", func(t *testing.T) {
		cfg := DefaultSecurityHeadersConfig
		WithSecurityHeadersExemptPaths(nil)(&cfg)
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := SecurityHeadersConfig{} // Zero values
		if cfg.ContentSecurityPolicy != "" {
			t.Errorf("expected zero CSP = '', got %s", cfg.ContentSecurityPolicy)
		}
		if cfg.ContentSecurityPolicyReportOnly != false {
			t.Errorf("expected zero CSP report only = false, got %t", cfg.ContentSecurityPolicyReportOnly)
		}
		if cfg.StrictTransportSecurity.MaxAge != 0 {
			t.Errorf("expected zero HSTS max age = 0, got %d", cfg.StrictTransportSecurity.MaxAge)
		}
		if cfg.ExemptPaths != nil {
			t.Errorf("expected zero exempt paths = nil, got %v", cfg.ExemptPaths)
		}
	})
}

func TestSecurityHeadersConfigToOptions(t *testing.T) {
	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy:           "default-src 'self'",
		ContentSecurityPolicyReportOnly: true,
		CrossOriginEmbedderPolicy:       "unsafe-none",
		CrossOriginOpenerPolicy:         "unsafe-none",
		CrossOriginResourcePolicy:       "cross-origin",
		PermissionsPolicy:               "camera=(), microphone=()",
		ReferrerPolicy:                  "strict-origin",
		Server:                          "TestServer/1.0",
		StrictTransportSecurity: StrictTransportSecurity{
			MaxAge:            31536000,
			ExcludeSubdomains: true,
			PreloadEnabled:    true,
		},
		XContentTypeOptions: "nosniff",
		XFrameOptions:       "SAMEORIGIN",
		ExemptPaths:         []string{"/public"},
	}

	options := securityHeadersConfigToOptions(cfg)
	if len(options) != 12 {
		t.Errorf("expected 12 options, got %d", len(options))
	}

	// Apply the options to a new config to test they work correctly
	newCfg := DefaultSecurityHeadersConfig
	for _, option := range options {
		option(&newCfg)
	}

	if newCfg.ContentSecurityPolicy != cfg.ContentSecurityPolicy {
		t.Error("expected CSP to be converted correctly")
	}
	if newCfg.ContentSecurityPolicyReportOnly != cfg.ContentSecurityPolicyReportOnly {
		t.Error("expected CSP report only to be converted correctly")
	}
	if newCfg.StrictTransportSecurity.MaxAge != cfg.StrictTransportSecurity.MaxAge {
		t.Error("expected HSTS max age to be converted correctly")
	}
	if !reflect.DeepEqual(newCfg.ExemptPaths, cfg.ExemptPaths) {
		t.Error("expected exempt paths to be converted correctly")
	}
}

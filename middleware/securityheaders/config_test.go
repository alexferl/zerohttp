package securityheaders

import (
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestSecurityHeadersConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	expectedCSP := "default-src 'none'; script-src 'self'; connect-src 'self'; img-src 'self'; style-src 'self'; frame-ancestors 'self'; form-action 'self';"
	zhtest.AssertEqual(t, expectedCSP, cfg.ContentSecurityPolicy)
	zhtest.AssertFalse(t, cfg.ContentSecurityPolicyReportOnly)
	zhtest.AssertEqual(t, "require-corp", cfg.CrossOriginEmbedderPolicy)
	zhtest.AssertEqual(t, "same-origin", cfg.CrossOriginOpenerPolicy)
	zhtest.AssertEqual(t, "same-origin", cfg.CrossOriginResourcePolicy)
	zhtest.AssertNotEmpty(t, cfg.PermissionsPolicy)
	zhtest.AssertEqual(t, "no-referrer", cfg.ReferrerPolicy)
	zhtest.AssertEqual(t, "", cfg.Server)
	zhtest.AssertEqual(t, "nosniff", cfg.XContentTypeOptions)
	zhtest.AssertEqual(t, "DENY", cfg.XFrameOptions)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

	// Test default HSTS values
	zhtest.AssertEqual(t, 0, cfg.StrictTransportSecurity.MaxAge)
	zhtest.AssertFalse(t, cfg.StrictTransportSecurity.ExcludeSubdomains)
	zhtest.AssertFalse(t, cfg.StrictTransportSecurity.PreloadEnabled)

	// Test permissions policy
	expectedPolicy := strings.Join(permissionPolicyFeatures, ", ")
	zhtest.AssertEqual(t, expectedPolicy, cfg.PermissionsPolicy)
}

func TestStrictTransportSecurity_DefaultValues(t *testing.T) {
	hsts := DefaultStrictTransportSecurity
	zhtest.AssertEqual(t, 0, hsts.MaxAge)
	zhtest.AssertFalse(t, hsts.ExcludeSubdomains)
	zhtest.AssertFalse(t, hsts.PreloadEnabled)
}

func TestPermissionPolicyFeatures(t *testing.T) {
	zhtest.AssertTrue(t, len(permissionPolicyFeatures) > 0)

	// Test feature format
	for _, feature := range permissionPolicyFeatures {
		zhtest.AssertTrue(t, strings.HasSuffix(feature, "=()"))
	}

	// Test specific expected features
	expectedFeatures := []string{"camera=()", "microphone=()", "geolocation=()", "fullscreen=()", "payment=()"}
	featureMap := make(map[string]bool)
	for _, feature := range permissionPolicyFeatures {
		featureMap[feature] = true
	}
	for _, expected := range expectedFeatures {
		zhtest.AssertTrue(t, featureMap[expected])
	}
}

func TestSecurityHeadersConfig_StructAssignment(t *testing.T) {
	t.Run("basic header fields", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           "default-src 'self'; script-src 'self' 'unsafe-inline'",
			ContentSecurityPolicyReportOnly: true,
			CrossOriginEmbedderPolicy:       "unsafe-none",
			CrossOriginOpenerPolicy:         "unsafe-none",
			CrossOriginResourcePolicy:       "cross-origin",
			PermissionsPolicy:               "camera=(), microphone=()",
			ReferrerPolicy:                  "strict-origin",
			Server:                          "nginx/1.18.0",
			StrictTransportSecurity: StrictTransportSecurity{
				MaxAge:            31536000,
				ExcludeSubdomains: true,
				PreloadEnabled:    true,
			},
			XContentTypeOptions: "nosniff",
			XFrameOptions:       "SAMEORIGIN",
			ExcludedPaths:       []string{},
		}

		zhtest.AssertEqual(t, "default-src 'self'; script-src 'self' 'unsafe-inline'", cfg.ContentSecurityPolicy)
		zhtest.AssertTrue(t, cfg.ContentSecurityPolicyReportOnly)
		zhtest.AssertEqual(t, "unsafe-none", cfg.CrossOriginEmbedderPolicy)
		zhtest.AssertEqual(t, "unsafe-none", cfg.CrossOriginOpenerPolicy)
		zhtest.AssertEqual(t, "cross-origin", cfg.CrossOriginResourcePolicy)
	})

	t.Run("policy and server fields", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               "camera=(), microphone=(), geolocation=()",
			ReferrerPolicy:                  "strict-origin",
			Server:                          "nginx/1.18.0",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "SAMEORIGIN",
			ExcludedPaths:                   []string{},
		}

		zhtest.AssertEqual(t, "camera=(), microphone=(), geolocation=()", cfg.PermissionsPolicy)
		zhtest.AssertEqual(t, "strict-origin", cfg.ReferrerPolicy)
		zhtest.AssertEqual(t, "nginx/1.18.0", cfg.Server)
		zhtest.AssertEqual(t, "nosniff", cfg.XContentTypeOptions)
		zhtest.AssertEqual(t, "SAMEORIGIN", cfg.XFrameOptions)
	})

	t.Run("HSTS fields", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity: StrictTransportSecurity{
				MaxAge:            31536000,
				ExcludeSubdomains: true,
				PreloadEnabled:    true,
			},
			XContentTypeOptions: "nosniff",
			XFrameOptions:       "DENY",
			ExcludedPaths:       []string{},
		}

		zhtest.AssertEqual(t, 31536000, cfg.StrictTransportSecurity.MaxAge)
		zhtest.AssertTrue(t, cfg.StrictTransportSecurity.ExcludeSubdomains)
		zhtest.AssertTrue(t, cfg.StrictTransportSecurity.PreloadEnabled)
	})

	t.Run("excluded paths field", func(t *testing.T) {
		excludedPaths := []string{"/api/webhook", "/health", "/metrics"}
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "DENY",
			ExcludedPaths:                   excludedPaths,
		}
		zhtest.AssertEqual(t, 3, len(cfg.ExcludedPaths))
		zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	})

	t.Run("included paths field", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "DENY",
			IncludedPaths:                   includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
	})
}

func TestSecurityHeadersConfig_MultipleFields(t *testing.T) {
	excludedPaths := []string{"/public", "/api/webhook"}
	includedPaths := []string{"/api/public"}
	cfg := Config{
		ContentSecurityPolicy:           "default-src 'self'",
		ContentSecurityPolicyReportOnly: true,
		CrossOriginEmbedderPolicy:       "unsafe-none",
		CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
		CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
		PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
		ReferrerPolicy:                  "strict-origin",
		Server:                          "MyServer/1.0",
		StrictTransportSecurity:         DefaultStrictTransportSecurity,
		XContentTypeOptions:             "nosniff",
		XFrameOptions:                   "SAMEORIGIN",
		ExcludedPaths:                   excludedPaths,
		IncludedPaths:                   includedPaths,
	}

	zhtest.AssertEqual(t, "default-src 'self'", cfg.ContentSecurityPolicy)
	zhtest.AssertTrue(t, cfg.ContentSecurityPolicyReportOnly)
	zhtest.AssertEqual(t, "unsafe-none", cfg.CrossOriginEmbedderPolicy)
	zhtest.AssertEqual(t, "strict-origin", cfg.ReferrerPolicy)
	zhtest.AssertEqual(t, "MyServer/1.0", cfg.Server)
	zhtest.AssertEqual(t, "SAMEORIGIN", cfg.XFrameOptions)
	zhtest.AssertDeepEqual(t, excludedPaths, cfg.ExcludedPaths)
	zhtest.AssertDeepEqual(t, includedPaths, cfg.IncludedPaths)
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
				cfg := Config{
					ContentSecurityPolicy:           tt.csp,
					ContentSecurityPolicyReportOnly: false,
					CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
					CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
					CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
					PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
					ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
					Server:                          "",
					StrictTransportSecurity:         DefaultStrictTransportSecurity,
					XContentTypeOptions:             "nosniff",
					XFrameOptions:                   "DENY",
					ExcludedPaths:                   []string{},
				}
				zhtest.AssertEqual(t, tt.csp, cfg.ContentSecurityPolicy)
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
				cfg := Config{
					ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
					ContentSecurityPolicyReportOnly: false,
					CrossOriginEmbedderPolicy:       tt.coep,
					CrossOriginOpenerPolicy:         tt.coop,
					CrossOriginResourcePolicy:       tt.corp,
					PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
					ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
					Server:                          "",
					StrictTransportSecurity:         DefaultStrictTransportSecurity,
					XContentTypeOptions:             "nosniff",
					XFrameOptions:                   "DENY",
					ExcludedPaths:                   []string{},
				}

				zhtest.AssertEqual(t, tt.coep, cfg.CrossOriginEmbedderPolicy)
				zhtest.AssertEqual(t, tt.coop, cfg.CrossOriginOpenerPolicy)
				zhtest.AssertEqual(t, tt.corp, cfg.CrossOriginResourcePolicy)
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
				cfg := Config{
					ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
					ContentSecurityPolicyReportOnly: false,
					CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
					CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
					CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
					PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
					ReferrerPolicy:                  policy,
					Server:                          "",
					StrictTransportSecurity:         DefaultStrictTransportSecurity,
					XContentTypeOptions:             "nosniff",
					XFrameOptions:                   "DENY",
					ExcludedPaths:                   []string{},
				}
				zhtest.AssertEqual(t, policy, cfg.ReferrerPolicy)
			})
		}
	})

	t.Run("X-Frame-Options values", func(t *testing.T) {
		options := []string{"DENY", "SAMEORIGIN", "ALLOW-FROM https://example.com", ""}

		for _, option := range options {
			t.Run(option, func(t *testing.T) {
				cfg := Config{
					ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
					ContentSecurityPolicyReportOnly: false,
					CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
					CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
					CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
					PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
					ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
					Server:                          "",
					StrictTransportSecurity:         DefaultStrictTransportSecurity,
					XContentTypeOptions:             "nosniff",
					XFrameOptions:                   option,
					ExcludedPaths:                   []string{},
				}
				zhtest.AssertEqual(t, option, cfg.XFrameOptions)
			})
		}
	})
}

func TestSecurityHeadersConfig_EdgeCases(t *testing.T) {
	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "DENY",
			ExcludedPaths:                   []string{},
		}
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "DENY",
			ExcludedPaths:                   nil,
		}
		zhtest.AssertNil(t, cfg.ExcludedPaths)
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "DENY",
			IncludedPaths:                   []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			ContentSecurityPolicy:           DefaultConfig.ContentSecurityPolicy,
			ContentSecurityPolicyReportOnly: false,
			CrossOriginEmbedderPolicy:       DefaultConfig.CrossOriginEmbedderPolicy,
			CrossOriginOpenerPolicy:         DefaultConfig.CrossOriginOpenerPolicy,
			CrossOriginResourcePolicy:       DefaultConfig.CrossOriginResourcePolicy,
			PermissionsPolicy:               DefaultConfig.PermissionsPolicy,
			ReferrerPolicy:                  DefaultConfig.ReferrerPolicy,
			Server:                          "",
			StrictTransportSecurity:         DefaultStrictTransportSecurity,
			XContentTypeOptions:             "nosniff",
			XFrameOptions:                   "DENY",
			IncludedPaths:                   nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})

	t.Run("zero values", func(t *testing.T) {
		cfg := Config{} // Zero values
		zhtest.AssertEqual(t, "", cfg.ContentSecurityPolicy)
		zhtest.AssertFalse(t, cfg.ContentSecurityPolicyReportOnly)
		zhtest.AssertEqual(t, 0, cfg.StrictTransportSecurity.MaxAge)
		zhtest.AssertNil(t, cfg.ExcludedPaths)
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})
}

func TestSecurityHeadersConfig_StructCreation(t *testing.T) {
	cfg := Config{
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
		ExcludedPaths:       []string{"/public"},
	}

	zhtest.AssertEqual(t, "default-src 'self'", cfg.ContentSecurityPolicy)
	zhtest.AssertTrue(t, cfg.ContentSecurityPolicyReportOnly)
	zhtest.AssertEqual(t, 31536000, cfg.StrictTransportSecurity.MaxAge)
	zhtest.AssertDeepEqual(t, []string{"/public"}, cfg.ExcludedPaths)
}

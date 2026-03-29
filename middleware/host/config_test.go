package host

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestHostValidationConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig

	zhtest.AssertEqual(t, 0, len(cfg.AllowedHosts))
	zhtest.AssertFalse(t, cfg.AllowSubdomains)
	zhtest.AssertFalse(t, cfg.StrictPort)
	zhtest.AssertEqual(t, http.StatusBadRequest, cfg.StatusCode)
	zhtest.AssertEqual(t, "Invalid Host header", cfg.Message)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
}

func TestHostValidationConfig_CustomValues(t *testing.T) {
	cfg := Config{
		AllowedHosts:    []string{"api.example.com", "example.com"},
		AllowSubdomains: true,
		StrictPort:      true,
		StatusCode:      http.StatusForbidden,
		Message:         "Forbidden host",
		ExcludedPaths:   []string{"/health", "/metrics"},
		IncludedPaths:   []string{"/api/public"},
	}

	zhtest.AssertEqual(t, 2, len(cfg.AllowedHosts))
	zhtest.AssertEqual(t, "api.example.com", cfg.AllowedHosts[0])
	zhtest.AssertEqual(t, "example.com", cfg.AllowedHosts[1])
	zhtest.AssertTrue(t, cfg.AllowSubdomains)
	zhtest.AssertTrue(t, cfg.StrictPort)
	zhtest.AssertEqual(t, http.StatusForbidden, cfg.StatusCode)
	zhtest.AssertEqual(t, "Forbidden host", cfg.Message)
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
}

func TestHostValidationConfig_PartialConfig(t *testing.T) {
	cfg := Config{
		AllowedHosts: []string{"example.com"},
		StatusCode:   http.StatusUnauthorized,
	}

	zhtest.AssertEqual(t, 1, len(cfg.AllowedHosts))
	zhtest.AssertEqual(t, http.StatusUnauthorized, cfg.StatusCode)
	// Unset fields should have zero values
	zhtest.AssertFalse(t, cfg.AllowSubdomains)
	zhtest.AssertEqual(t, "", cfg.Message)
}

func TestHostValidationConfig_EmptyAllowedHosts(t *testing.T) {
	// Empty AllowedHosts means validation is disabled
	cfg := Config{
		AllowedHosts: []string{},
	}

	zhtest.AssertEqual(t, 0, len(cfg.AllowedHosts))
}

func TestHostValidationConfig_NilAllowedHosts(t *testing.T) {
	cfg := Config{
		AllowedHosts: nil,
	}

	zhtest.AssertNil(t, cfg.AllowedHosts)
}

func TestHostValidationConfig_StatusCodeOptions(t *testing.T) {
	testCases := []int{
		http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
		0,
	}

	for _, code := range testCases {
		cfg := Config{
			StatusCode: code,
		}
		zhtest.AssertEqual(t, code, cfg.StatusCode)
	}
}

func TestHostValidationConfig_ExcludedPaths(t *testing.T) {
	tests := []struct {
		name          string
		excludedPaths []string
		expectedLen   int
		expectedPath  string
	}{
		{
			name:          "health and metrics",
			excludedPaths: []string{"/health", "/metrics"},
			expectedLen:   2,
			expectedPath:  "/health",
		},
		{
			name:          "single path",
			excludedPaths: []string{"/api/status"},
			expectedLen:   1,
			expectedPath:  "/api/status",
		},
		{
			name:          "wildcard paths",
			excludedPaths: []string{"/health/*", "/public/*"},
			expectedLen:   2,
			expectedPath:  "/health/*",
		},
		{
			name:          "empty paths",
			excludedPaths: []string{},
			expectedLen:   0,
			expectedPath:  "",
		},
		{
			name:          "nil paths",
			excludedPaths: nil,
			expectedLen:   0,
			expectedPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ExcludedPaths: tt.excludedPaths,
			}
			zhtest.AssertEqual(t, tt.expectedLen, len(cfg.ExcludedPaths))
			if tt.expectedLen > 0 {
				zhtest.AssertEqual(t, tt.expectedPath, cfg.ExcludedPaths[0])
			}
		})
	}
}

func TestHostValidationConfig_IncludedPaths(t *testing.T) {
	tests := []struct {
		name          string
		includedPaths []string
		expectedLen   int
		expectedPath  string
	}{
		{
			name:          "public paths",
			includedPaths: []string{"/api/public", "/health"},
			expectedLen:   2,
			expectedPath:  "/api/public",
		},
		{
			name:          "single path",
			includedPaths: []string{"/api/status"},
			expectedLen:   1,
			expectedPath:  "/api/status",
		},
		{
			name:          "wildcard paths",
			includedPaths: []string{"/public/*", "/api/v1/*"},
			expectedLen:   2,
			expectedPath:  "/public/*",
		},
		{
			name:          "empty paths",
			includedPaths: []string{},
			expectedLen:   0,
			expectedPath:  "",
		},
		{
			name:          "nil paths",
			includedPaths: nil,
			expectedLen:   0,
			expectedPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				IncludedPaths: tt.includedPaths,
			}
			zhtest.AssertEqual(t, tt.expectedLen, len(cfg.IncludedPaths))
			if tt.expectedLen > 0 {
				zhtest.AssertEqual(t, tt.expectedPath, cfg.IncludedPaths[0])
			}
		})
	}
}

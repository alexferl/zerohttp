package config

import (
	"net/http"
	"testing"
)

func TestHostValidationConfig_DefaultValues(t *testing.T) {
	cfg := DefaultHostValidationConfig

	if len(cfg.AllowedHosts) != 0 {
		t.Errorf("expected default AllowedHosts to be empty, got %d hosts", len(cfg.AllowedHosts))
	}
	if cfg.AllowSubdomains != false {
		t.Errorf("expected default AllowSubdomains to be false, got %t", cfg.AllowSubdomains)
	}
	if cfg.StrictPort != false {
		t.Errorf("expected default StrictPort to be false, got %t", cfg.StrictPort)
	}
	if cfg.StatusCode != http.StatusBadRequest {
		t.Errorf("expected default StatusCode to be %d, got %d", http.StatusBadRequest, cfg.StatusCode)
	}
	if cfg.Message != "Invalid Host header" {
		t.Errorf("expected default Message to be 'Invalid Host header', got '%s'", cfg.Message)
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default ExcludedPaths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected default IncludedPaths to be empty, got %d paths", len(cfg.IncludedPaths))
	}
}

func TestHostValidationConfig_CustomValues(t *testing.T) {
	cfg := HostValidationConfig{
		AllowedHosts:    []string{"api.example.com", "example.com"},
		AllowSubdomains: true,
		StrictPort:      true,
		StatusCode:      http.StatusForbidden,
		Message:         "Forbidden host",
		ExcludedPaths:   []string{"/health", "/metrics"},
		IncludedPaths:   []string{"/api/public"},
	}

	if len(cfg.AllowedHosts) != 2 {
		t.Errorf("expected 2 allowed hosts, got %d", len(cfg.AllowedHosts))
	}
	if cfg.AllowedHosts[0] != "api.example.com" {
		t.Errorf("expected first host to be 'api.example.com', got '%s'", cfg.AllowedHosts[0])
	}
	if cfg.AllowedHosts[1] != "example.com" {
		t.Errorf("expected second host to be 'example.com', got '%s'", cfg.AllowedHosts[1])
	}
	if !cfg.AllowSubdomains {
		t.Error("expected AllowSubdomains to be true")
	}
	if !cfg.StrictPort {
		t.Error("expected StrictPort to be true")
	}
	if cfg.StatusCode != http.StatusForbidden {
		t.Errorf("expected StatusCode to be %d, got %d", http.StatusForbidden, cfg.StatusCode)
	}
	if cfg.Message != "Forbidden host" {
		t.Errorf("expected Message to be 'Forbidden host', got '%s'", cfg.Message)
	}
	if len(cfg.ExcludedPaths) != 2 {
		t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 1 {
		t.Errorf("expected 1 allowed path, got %d", len(cfg.IncludedPaths))
	}
}

func TestHostValidationConfig_PartialConfig(t *testing.T) {
	cfg := HostValidationConfig{
		AllowedHosts: []string{"example.com"},
		StatusCode:   http.StatusUnauthorized,
	}

	if len(cfg.AllowedHosts) != 1 {
		t.Errorf("expected 1 allowed host, got %d", len(cfg.AllowedHosts))
	}
	if cfg.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected StatusCode to be %d, got %d", http.StatusUnauthorized, cfg.StatusCode)
	}
	// Unset fields should have zero values
	if cfg.AllowSubdomains {
		t.Error("expected AllowSubdomains to be false (zero value)")
	}
	if cfg.Message != "" {
		t.Errorf("expected Message to be empty (zero value), got '%s'", cfg.Message)
	}
}

func TestHostValidationConfig_EmptyAllowedHosts(t *testing.T) {
	// Empty AllowedHosts means validation is disabled
	cfg := HostValidationConfig{
		AllowedHosts: []string{},
	}

	if len(cfg.AllowedHosts) != 0 {
		t.Errorf("expected AllowedHosts to be empty, got %d", len(cfg.AllowedHosts))
	}
}

func TestHostValidationConfig_NilAllowedHosts(t *testing.T) {
	cfg := HostValidationConfig{
		AllowedHosts: nil,
	}

	if cfg.AllowedHosts != nil {
		t.Error("expected AllowedHosts to be nil")
	}
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
		cfg := HostValidationConfig{
			StatusCode: code,
		}
		if cfg.StatusCode != code {
			t.Errorf("expected StatusCode %d, got %d", code, cfg.StatusCode)
		}
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
			cfg := HostValidationConfig{
				ExcludedPaths: tt.excludedPaths,
			}
			if len(cfg.ExcludedPaths) != tt.expectedLen {
				t.Errorf("expected %d excluded paths, got %d", tt.expectedLen, len(cfg.ExcludedPaths))
			}
			if tt.expectedLen > 0 && cfg.ExcludedPaths[0] != tt.expectedPath {
				t.Errorf("expected first path to be '%s', got '%s'", tt.expectedPath, cfg.ExcludedPaths[0])
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
			cfg := HostValidationConfig{
				IncludedPaths: tt.includedPaths,
			}
			if len(cfg.IncludedPaths) != tt.expectedLen {
				t.Errorf("expected %d included paths, got %d", tt.expectedLen, len(cfg.IncludedPaths))
			}
			if tt.expectedLen > 0 && cfg.IncludedPaths[0] != tt.expectedPath {
				t.Errorf("expected first path to be '%s', got '%s'", tt.expectedPath, cfg.IncludedPaths[0])
			}
		})
	}
}

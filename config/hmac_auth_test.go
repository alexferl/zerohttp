package config

import (
	"net/http"
	"testing"
	"time"
)

func TestHMACAuthConfig_DefaultValues(t *testing.T) {
	cfg := DefaultHMACAuthConfig

	if cfg.Algorithm != HMACSHA256 {
		t.Errorf("expected default algorithm to be HMACSHA256, got %s", cfg.Algorithm)
	}

	if cfg.MaxSkew != 5*time.Minute {
		t.Errorf("expected default MaxSkew to be 5 minutes, got %v", cfg.MaxSkew)
	}

	if cfg.ClockSkewGrace != 1*time.Minute {
		t.Errorf("expected default ClockSkewGrace to be 1 minute, got %v", cfg.ClockSkewGrace)
	}

	if cfg.AuthHeaderName != "Authorization" {
		t.Errorf("expected default AuthHeaderName to be 'Authorization', got %s", cfg.AuthHeaderName)
	}

	if cfg.TimestampHeader != "X-Timestamp" {
		t.Errorf("expected default TimestampHeader to be 'X-Timestamp', got %s", cfg.TimestampHeader)
	}

	if cfg.AllowUnsignedPayload != false {
		t.Error("expected default AllowUnsignedPayload to be false")
	}

	if cfg.CredentialStore != nil {
		t.Error("expected default CredentialStore to be nil")
	}
}

func TestHMACHashAlgorithm_String(t *testing.T) {
	tests := []struct {
		algo HMACHashAlgorithm
		want string
	}{
		{HMACSHA256, "SHA256"},
		{HMACSHA384, "SHA384"},
		{HMACSHA512, "SHA512"},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			if string(tt.algo) != tt.want {
				t.Errorf("got %s, want %s", tt.algo, tt.want)
			}
		})
	}
}

func TestHMACAuthConfig_CustomValues(t *testing.T) {
	customStore := func(id string) []string { return []string{"secret"} }
	customErrorHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}

	cfg := HMACAuthConfig{
		CredentialStore:      customStore,
		Algorithm:            HMACSHA512,
		MaxSkew:              10 * time.Minute,
		ClockSkewGrace:       2 * time.Minute,
		RequiredHeaders:      []string{"host", "x-timestamp", "x-request-id"},
		OptionalHeaders:      []string{"content-type", "x-correlation-id"},
		ExemptPaths:          []string{"/health", "/metrics"},
		ErrorHandler:         customErrorHandler,
		AuthHeaderName:       "X-Authorization",
		TimestampHeader:      "X-Date",
		AllowUnsignedPayload: true,
	}

	if cfg.Algorithm != HMACSHA512 {
		t.Errorf("expected algorithm to be HMACSHA512, got %s", cfg.Algorithm)
	}

	if cfg.MaxSkew != 10*time.Minute {
		t.Errorf("expected MaxSkew to be 10 minutes, got %v", cfg.MaxSkew)
	}

	if cfg.ClockSkewGrace != 2*time.Minute {
		t.Errorf("expected ClockSkewGrace to be 2 minutes, got %v", cfg.ClockSkewGrace)
	}

	if cfg.AuthHeaderName != "X-Authorization" {
		t.Errorf("expected AuthHeaderName to be 'X-Authorization', got %s", cfg.AuthHeaderName)
	}

	if cfg.TimestampHeader != "X-Date" {
		t.Errorf("expected TimestampHeader to be 'X-Date', got %s", cfg.TimestampHeader)
	}

	if !cfg.AllowUnsignedPayload {
		t.Error("expected AllowUnsignedPayload to be true")
	}

	if cfg.CredentialStore == nil {
		t.Error("expected CredentialStore to be set")
	}

	// Test CredentialStore works
	secrets := cfg.CredentialStore("test-key")
	if len(secrets) != 1 || secrets[0] != "secret" {
		t.Errorf("expected CredentialStore to return ['secret'], got %v", secrets)
	}

	if cfg.ErrorHandler == nil {
		t.Error("expected ErrorHandler to be set")
	}

	if len(cfg.ExemptPaths) != 2 {
		t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
	}

	if len(cfg.RequiredHeaders) != 3 {
		t.Errorf("expected 3 required headers, got %d", len(cfg.RequiredHeaders))
	}
}

func TestHMACAuthConfig_EmptySlices(t *testing.T) {
	cfg := HMACAuthConfig{
		CredentialStore: func(id string) []string { return nil },
		RequiredHeaders: []string{},
		OptionalHeaders: []string{},
		ExemptPaths:     []string{},
	}

	if cfg.RequiredHeaders == nil {
		t.Error("RequiredHeaders should not be nil, should be empty slice")
	}

	if cfg.OptionalHeaders == nil {
		t.Error("OptionalHeaders should not be nil, should be empty slice")
	}

	if cfg.ExemptPaths == nil {
		t.Error("ExemptPaths should not be nil, should be empty slice")
	}
}

func TestHMACAuthConfig_AllAlgorithms(t *testing.T) {
	algorithms := []HMACHashAlgorithm{HMACSHA256, HMACSHA384, HMACSHA512}

	for _, algo := range algorithms {
		t.Run(string(algo), func(t *testing.T) {
			cfg := HMACAuthConfig{
				CredentialStore: func(id string) []string { return []string{"secret"} },
				Algorithm:       algo,
			}

			if cfg.Algorithm != algo {
				t.Errorf("expected algorithm %s, got %s", algo, cfg.Algorithm)
			}
		})
	}
}

func TestHMACAuthConfig_ZeroDuration(t *testing.T) {
	cfg := HMACAuthConfig{
		CredentialStore: func(id string) []string { return nil },
		MaxSkew:         0,
		ClockSkewGrace:  0,
	}

	// Zero durations should be overwritten by defaults in middleware
	if cfg.MaxSkew != 0 {
		t.Error("MaxSkew should be 0")
	}

	if cfg.ClockSkewGrace != 0 {
		t.Error("ClockSkewGrace should be 0")
	}
}

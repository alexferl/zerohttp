package hmacauth

import (
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestHMACAuthConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig

	zhtest.AssertEqual(t, SHA256, cfg.Algorithm)
	zhtest.AssertEqual(t, 5*time.Minute, cfg.MaxSkew)
	zhtest.AssertEqual(t, 1*time.Minute, cfg.ClockSkewGrace)
	zhtest.AssertEqual(t, "Authorization", cfg.AuthHeaderName)
	zhtest.AssertEqual(t, "X-Timestamp", cfg.TimestampHeader)
	zhtest.AssertFalse(t, cfg.AllowUnsignedPayload)
	zhtest.AssertNil(t, cfg.CredentialStore)
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
}

func TestHMACHashAlgorithm_String(t *testing.T) {
	tests := []struct {
		algo HashAlgorithm
		want string
	}{
		{SHA256, "SHA256"},
		{SHA384, "SHA384"},
		{SHA512, "SHA512"},
	}

	for _, tt := range tests {
		t.Run(string(tt.algo), func(t *testing.T) {
			zhtest.AssertEqual(t, tt.want, string(tt.algo))
		})
	}
}

func TestHMACAuthConfig_CustomValues(t *testing.T) {
	customStore := func(id string) []string { return []string{"secret"} }
	customErrorHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}

	cfg := Config{
		CredentialStore:      customStore,
		Algorithm:            SHA512,
		MaxSkew:              10 * time.Minute,
		ClockSkewGrace:       2 * time.Minute,
		RequiredHeaders:      []string{"host", "x-timestamp", "x-request-id"},
		OptionalHeaders:      []string{"content-type", "x-correlation-id"},
		ExcludedPaths:        []string{"/health", "/metrics"},
		IncludedPaths:        []string{"/api/public"},
		ErrorHandler:         customErrorHandler,
		AuthHeaderName:       "X-Authorization",
		TimestampHeader:      "X-Date",
		AllowUnsignedPayload: true,
	}

	zhtest.AssertEqual(t, SHA512, cfg.Algorithm)
	zhtest.AssertEqual(t, 10*time.Minute, cfg.MaxSkew)
	zhtest.AssertEqual(t, 2*time.Minute, cfg.ClockSkewGrace)
	zhtest.AssertEqual(t, "X-Authorization", cfg.AuthHeaderName)
	zhtest.AssertEqual(t, "X-Date", cfg.TimestampHeader)
	zhtest.AssertTrue(t, cfg.AllowUnsignedPayload)
	zhtest.AssertNotNil(t, cfg.CredentialStore)

	// Test CredentialStore works
	secrets := cfg.CredentialStore("test-key")
	zhtest.AssertEqual(t, 1, len(secrets))
	zhtest.AssertEqual(t, "secret", secrets[0])

	zhtest.AssertNotNil(t, cfg.ErrorHandler)
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 1, len(cfg.IncludedPaths))
	zhtest.AssertEqual(t, 3, len(cfg.RequiredHeaders))
}

func TestHMACAuthConfig_EmptySlices(t *testing.T) {
	cfg := Config{
		CredentialStore: func(id string) []string { return nil },
		RequiredHeaders: []string{},
		OptionalHeaders: []string{},
		ExcludedPaths:   []string{},
		IncludedPaths:   []string{},
	}

	zhtest.AssertNotNil(t, cfg.RequiredHeaders)
	zhtest.AssertNotNil(t, cfg.OptionalHeaders)
	zhtest.AssertNotNil(t, cfg.ExcludedPaths)
	zhtest.AssertNotNil(t, cfg.IncludedPaths)
}

func TestHMACAuthConfig_AllAlgorithms(t *testing.T) {
	algorithms := []HashAlgorithm{SHA256, SHA384, SHA512}

	for _, algo := range algorithms {
		t.Run(string(algo), func(t *testing.T) {
			cfg := Config{
				CredentialStore: func(id string) []string { return []string{"secret"} },
				Algorithm:       algo,
			}

			zhtest.AssertEqual(t, algo, cfg.Algorithm)
		})
	}
}

func TestHMACAuthConfig_ZeroDuration(t *testing.T) {
	cfg := Config{
		CredentialStore: func(id string) []string { return nil },
		MaxSkew:         0,
		ClockSkewGrace:  0,
	}

	// Zero durations should be overwritten by defaults in middleware
	zhtest.AssertEqual(t, time.Duration(0), cfg.MaxSkew)
	zhtest.AssertEqual(t, time.Duration(0), cfg.ClockSkewGrace)
}

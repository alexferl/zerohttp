package jwtauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

func TestDefaultJWTAuthConfig(t *testing.T) {
	cfg := DefaultConfig

	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("expected AccessTokenTTL to be 15m, got %v", cfg.AccessTokenTTL)
	}

	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("expected RefreshTokenTTL to be 7d, got %v", cfg.RefreshTokenTTL)
	}

	if cfg.Extractor == nil {
		t.Error("expected TokenExtractor to be set")
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "valid bearer token",
			header: "Bearer abc123",
			want:   "abc123",
		},
		{
			name:   "valid bearer token with extra spaces",
			header: "Bearer  abc123  ",
			want:   "abc123",
		},
		{
			name:   "missing bearer prefix",
			header: "abc123",
			want:   "",
		},
		{
			name:   "empty header",
			header: "",
			want:   "",
		},
		{
			name:   "wrong prefix",
			header: "Basic abc123",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set(httpx.HeaderAuthorization, tt.header)
			}

			got := extractBearerToken(req)
			if got != tt.want {
				t.Errorf("extractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTokenType(t *testing.T) {
	if AccessToken != 0 {
		t.Errorf("expected AccessToken to be 0, got %d", AccessToken)
	}
	if RefreshToken != 1 {
		t.Errorf("expected RefreshToken to be 1, got %d", RefreshToken)
	}
}

func TestJWTAuthConfig_Customization(t *testing.T) {
	customExtractor := func(r *http.Request) string {
		return r.Header.Get("X-Custom-Token")
	}

	cfg := Config{
		Extractor:       customExtractor,
		RequiredClaims:  []string{"sub", "iss"},
		ExcludedPaths:   []string{"/health", "/metrics"},
		ExcludedMethods: []string{http.MethodHead},
		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}

	// Test custom extractor
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Token", "my-token")
	if got := cfg.Extractor(req); got != "my-token" {
		t.Errorf("expected token 'my-token', got %q", got)
	}

	if len(cfg.RequiredClaims) != 2 || cfg.RequiredClaims[0] != "sub" || cfg.RequiredClaims[1] != "iss" {
		t.Errorf("expected RequiredClaims [sub iss], got %v", cfg.RequiredClaims)
	}

	if len(cfg.ExcludedPaths) != 2 || cfg.ExcludedPaths[0] != "/health" || cfg.ExcludedPaths[1] != "/metrics" {
		t.Errorf("expected ExcludedPaths [/health /metrics], got %v", cfg.ExcludedPaths)
	}

	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected 0 included paths, got %d", len(cfg.IncludedPaths))
	}

	if len(cfg.ExcludedMethods) != 1 || cfg.ExcludedMethods[0] != http.MethodHead {
		t.Errorf("expected ExcludedMethods [HEAD], got %v", cfg.ExcludedMethods)
	}

	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Errorf("expected AccessTokenTTL to be 30m, got %v", cfg.AccessTokenTTL)
	}
}

func TestJWTAuthConfig_IncludedPaths(t *testing.T) {
	customExtractor := func(r *http.Request) string {
		return r.Header.Get("X-Custom-Token")
	}

	cfg := Config{
		Extractor:       customExtractor,
		RequiredClaims:  []string{"sub", "iss"},
		ExcludedPaths:   []string{"/health"},
		IncludedPaths:   []string{"/api/public", "/api/internal"},
		ExcludedMethods: []string{http.MethodHead},
		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}

	if len(cfg.IncludedPaths) != 2 {
		t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
	}
	if cfg.IncludedPaths[0] != "/api/public" {
		t.Errorf("expected first allowed path to be /api/public, got %s", cfg.IncludedPaths[0])
	}
	if cfg.IncludedPaths[1] != "/api/internal" {
		t.Errorf("expected second allowed path to be /api/internal, got %s", cfg.IncludedPaths[1])
	}

	// Test empty included paths
	cfg2 := Config{
		IncludedPaths: []string{},
	}
	if cfg2.IncludedPaths == nil {
		t.Error("expected included paths slice to be initialized, not nil")
	}
	if len(cfg2.IncludedPaths) != 0 {
		t.Errorf("expected empty included paths slice, got %d entries", len(cfg2.IncludedPaths))
	}

	// Test nil included paths
	cfg3 := Config{
		IncludedPaths: nil,
	}
	if cfg3.IncludedPaths != nil {
		t.Error("expected included paths to remain nil when nil is passed")
	}
}

func TestJWTClaimConstants(t *testing.T) {
	if JWTClaimSubject != "sub" {
		t.Errorf("expected JWTClaimSubject = sub, got %s", JWTClaimSubject)
	}
	if JWTClaimIssuer != "iss" {
		t.Errorf("expected JWTClaimIssuer = iss, got %s", JWTClaimIssuer)
	}
	if JWTClaimAudience != "aud" {
		t.Errorf("expected JWTClaimAudience = aud, got %s", JWTClaimAudience)
	}
	if JWTClaimExpiration != "exp" {
		t.Errorf("expected JWTClaimExpiration = exp, got %s", JWTClaimExpiration)
	}
	if JWTClaimNotBefore != "nbf" {
		t.Errorf("expected JWTClaimNotBefore = nbf, got %s", JWTClaimNotBefore)
	}
	if JWTClaimIssuedAt != "iat" {
		t.Errorf("expected JWTClaimIssuedAt = iat, got %s", JWTClaimIssuedAt)
	}
	if JWTClaimJWTID != "jti" {
		t.Errorf("expected JWTClaimJWTID = jti, got %s", JWTClaimJWTID)
	}
	if JWTClaimScope != "scope" {
		t.Errorf("expected JWTClaimScope = scope, got %s", JWTClaimScope)
	}
	if JWTClaimType != "type" {
		t.Errorf("expected JWTClaimType = type, got %s", JWTClaimType)
	}
}

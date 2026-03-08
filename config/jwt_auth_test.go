package config

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultJWTAuthConfig(t *testing.T) {
	cfg := DefaultJWTAuthConfig

	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("expected AccessTokenTTL to be 15m, got %v", cfg.AccessTokenTTL)
	}

	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Errorf("expected RefreshTokenTTL to be 7d, got %v", cfg.RefreshTokenTTL)
	}

	if cfg.TokenExtractor == nil {
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
				req.Header.Set("Authorization", tt.header)
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

	cfg := JWTAuthConfig{
		TokenExtractor:  customExtractor,
		RequiredClaims:  []string{"sub", "iss"},
		ExemptPaths:     []string{"/health", "/metrics"},
		ExemptMethods:   []string{http.MethodHead},
		AccessTokenTTL:  30 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}

	// Test custom extractor
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Token", "my-token")
	if got := cfg.TokenExtractor(req); got != "my-token" {
		t.Errorf("expected token 'my-token', got %q", got)
	}

	if len(cfg.RequiredClaims) != 2 || cfg.RequiredClaims[0] != "sub" || cfg.RequiredClaims[1] != "iss" {
		t.Errorf("expected RequiredClaims [sub iss], got %v", cfg.RequiredClaims)
	}

	if len(cfg.ExemptPaths) != 2 || cfg.ExemptPaths[0] != "/health" || cfg.ExemptPaths[1] != "/metrics" {
		t.Errorf("expected ExemptPaths [/health /metrics], got %v", cfg.ExemptPaths)
	}

	if len(cfg.ExemptMethods) != 1 || cfg.ExemptMethods[0] != http.MethodHead {
		t.Errorf("expected ExemptMethods [HEAD], got %v", cfg.ExemptMethods)
	}

	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Errorf("expected AccessTokenTTL to be 30m, got %v", cfg.AccessTokenTTL)
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

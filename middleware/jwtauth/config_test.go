package jwtauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultJWTAuthConfig(t *testing.T) {
	cfg := DefaultConfig

	zhtest.AssertEqual(t, 15*time.Minute, cfg.AccessTokenTTL)
	zhtest.AssertEqual(t, 7*24*time.Hour, cfg.RefreshTokenTTL)
	zhtest.AssertNotNil(t, cfg.Extractor)
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
			zhtest.AssertEqual(t, tt.want, got)
		})
	}
}

func TestTokenType(t *testing.T) {
	zhtest.AssertEqual(t, 0, int(AccessToken))
	zhtest.AssertEqual(t, 1, int(RefreshToken))
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
	zhtest.AssertEqual(t, "my-token", cfg.Extractor(req))

	zhtest.AssertEqual(t, 2, len(cfg.RequiredClaims))
	zhtest.AssertEqual(t, "sub", cfg.RequiredClaims[0])
	zhtest.AssertEqual(t, "iss", cfg.RequiredClaims[1])

	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, "/health", cfg.ExcludedPaths[0])
	zhtest.AssertEqual(t, "/metrics", cfg.ExcludedPaths[1])

	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))

	zhtest.AssertEqual(t, 1, len(cfg.ExcludedMethods))
	zhtest.AssertEqual(t, http.MethodHead, cfg.ExcludedMethods[0])

	zhtest.AssertEqual(t, 30*time.Minute, cfg.AccessTokenTTL)
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

	zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
	zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
	zhtest.AssertEqual(t, "/api/internal", cfg.IncludedPaths[1])

	// Test empty included paths
	cfg2 := Config{
		IncludedPaths: []string{},
	}
	zhtest.AssertNotNil(t, cfg2.IncludedPaths)
	zhtest.AssertEqual(t, 0, len(cfg2.IncludedPaths))

	// Test nil included paths
	cfg3 := Config{
		IncludedPaths: nil,
	}
	zhtest.AssertNil(t, cfg3.IncludedPaths)
}

func TestJWTClaimConstants(t *testing.T) {
	zhtest.AssertEqual(t, "sub", JWTClaimSubject)
	zhtest.AssertEqual(t, "iss", JWTClaimIssuer)
	zhtest.AssertEqual(t, "aud", JWTClaimAudience)
	zhtest.AssertEqual(t, "exp", JWTClaimExpiration)
	zhtest.AssertEqual(t, "nbf", JWTClaimNotBefore)
	zhtest.AssertEqual(t, "iat", JWTClaimIssuedAt)
	zhtest.AssertEqual(t, "jti", JWTClaimJWTID)
	zhtest.AssertEqual(t, "scope", JWTClaimScope)
	zhtest.AssertEqual(t, "type", JWTClaimType)
}

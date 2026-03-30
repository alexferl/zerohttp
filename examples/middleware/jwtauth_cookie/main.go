package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/middleware/securityheaders"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

// TokenStore implements jwtauth.Store with HS256 and in-memory revocation.
// In production, use Redis or database for revocation storage.
type TokenStore struct {
	mu      sync.RWMutex
	revoked map[string]bool // map of jti -> revoked
	hs256   *jwtauth.HS256Store
	opts    jwtauth.HS256Config
}

// NewTokenStore creates a new TokenStore with HS256 and revocation support
func NewTokenStore(secret []byte, opts jwtauth.HS256Config) *TokenStore {
	return &TokenStore{
		revoked: make(map[string]bool),
		hs256:   jwtauth.NewHS256Store(secret, opts),
		opts:    opts,
	}
}

// Validate parses and validates a JWT token
func (s *TokenStore) Validate(ctx context.Context, token string) (jwtauth.JWTClaims, error) {
	return s.hs256.Validate(ctx, token)
}

// Generate creates a new JWT token
func (s *TokenStore) Generate(ctx context.Context, claims jwtauth.JWTClaims, tokenType jwtauth.TokenType, ttl time.Duration) (string, error) {
	return s.hs256.Generate(ctx, claims, tokenType, ttl)
}

// Revoke invalidates a refresh token by its jti claim
func (s *TokenStore) Revoke(_ context.Context, claims map[string]any) error {
	jti := getJTI(claims)
	if jti != "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.revoked[jti] = true
	}
	return nil
}

// IsRevoked checks if a refresh token has been revoked
func (s *TokenStore) IsRevoked(_ context.Context, claims map[string]any) (bool, error) {
	jti := getJTI(claims)
	if jti == "" {
		return false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revoked[jti], nil
}

// Close releases resources associated with the store
func (s *TokenStore) Close() error {
	return nil
}

// getJTI extracts the jti claim from claims
func getJTI(claims map[string]any) string {
	if jti, ok := claims["jti"].(string); ok {
		return jti
	}
	return ""
}

//go:embed static/index.html
var indexHTML string

func main() {
	app := zh.New()

	// Add security headers with CSP nonce generation
	app.Use(securityheaders.New(securityheaders.Config{
		ContentSecurityPolicyNonceEnabled: true,
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'nonce-{{nonce}}'; " +
			"style-src 'nonce-{{nonce}}'; " +
			"img-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none';",
	}))

	// HS256 configuration
	hp := jwtauth.HS256Config{
		Secret: jwtSecret,
		Issuer: "zerohttp-example",
	}

	// Create TokenStore with HS256 and revocation support
	tokenStore := NewTokenStore(jwtSecret, hp)

	jwtCfg := jwtauth.Config{
		Store:           tokenStore,
		RequiredClaims:  []string{"sub"},
		ExcludedPaths:   []string{"/login", "/register", "/auth/refresh", "/auth/logout"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		// Enable cookie support
		// Uses default cookie names: access_token (Path: /) and refresh_token (Path: /auth)
		Extractor: jwtauth.HeaderOrCookieExtractor("access_token"),
		Cookie: jwtauth.CookieConfig{
			Enabled:     true,
			Path:        "/",
			RefreshPath: "/auth",
			Secure:      false, // Set to true in production (HTTPS only)
			HttpOnly:    true,  // Prevents JavaScript access
			SameSite:    http.SameSiteStrictMode,
		},
	}

	// Serve HTML demo with CSP nonce
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nonce := securityheaders.GetCSPNonce(r)
		html := strings.ReplaceAll(indexHTML, "{{nonce}}", nonce)
		return zh.R.HTML(w, http.StatusOK, html)
	}))

	// Public endpoints
	app.POST("/login", loginHandler(jwtCfg))
	app.POST("/register", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "registration endpoint - implement your logic here",
		})
	}))

	// Token refresh and logout endpoints (handle cookies automatically)
	app.POST("/auth/refresh", jwtauth.RefreshTokenHandler(jwtCfg))
	app.POST("/auth/logout", jwtauth.LogoutTokenHandler(jwtCfg))

	// Protected endpoints
	app.Use(jwtauth.New(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(profileHandler))
	app.GET("/api/admin", zh.HandlerFunc(adminHandler))

	log.Fatal(app.Start())
}

// loginHandler authenticates users and sets JWT tokens
func loginHandler(cfg jwtauth.Config) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := zh.B.JSON(r.Body, &req); err != nil {
			return zh.R.JSON(w, http.StatusBadRequest, zh.M{"error": "invalid request"})
		}

		// In production, verify against database
		if req.Username != "alice" || req.Password != "secret" {
			return zh.R.JSON(w, http.StatusUnauthorized, zh.M{"error": "invalid credentials"})
		}

		// Create claims with jti for revocation tracking
		claims := jwtauth.HS256Claims{
			"sub": req.Username,
			"jti": fmt.Sprintf("%s-%d", req.Username, time.Now().Unix()),
		}

		// Generate access token
		accessToken, err := jwtauth.GenerateAccessToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate access token"})
		}

		// Generate refresh token
		refreshToken, err := jwtauth.GenerateRefreshToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate refresh token"})
		}

		// Set access token cookie (Path: /) and refresh token cookie (Path: /auth)
		// Both are HttpOnly - browser handles them automatically
		jwtauth.SetCookie(w, accessToken, cfg)
		jwtauth.SetRefreshCookie(w, refreshToken, cfg)

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "logged in successfully",
		})
	}
}

func profileHandler(w http.ResponseWriter, r *http.Request) error {
	jwt := jwtauth.GetClaims(r)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": jwt.Subject(),
		"scopes":  jwt.Scopes(),
		"message": "This is your profile",
	})
}

func adminHandler(w http.ResponseWriter, r *http.Request) error {
	if !jwtauth.GetClaims(r).HasScope("admin") {
		return zh.R.JSON(w, http.StatusForbidden, zh.M{"error": "admin scope required"})
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message": "Admin access granted",
	})
}

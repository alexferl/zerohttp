package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/jwtauth"
)

var refreshJWTSecret = []byte("your-secret-key-change-in-production")

// JWTTokenStore implements config.TokenStore with HS256 and in-memory revocation.
// In production, use Redis or database for revocation storage.
type JWTTokenStore struct {
	mu      sync.RWMutex
	revoked map[string]bool // map of jti -> revoked
	hs256   *jwtauth.HS256Store
	opts    jwtauth.HS256Config
}

// NewJWTTokenStore creates a new TokenStore with HS256 and revocation support
func NewJWTTokenStore(secret []byte, opts jwtauth.HS256Config) *JWTTokenStore {
	return &JWTTokenStore{
		revoked: make(map[string]bool),
		hs256:   jwtauth.NewHS256Store(secret, opts),
		opts:    opts,
	}
}

// Validate parses and validates a JWT token
func (s *JWTTokenStore) Validate(ctx context.Context, token string) (jwtauth.JWTClaims, error) {
	return s.hs256.Validate(ctx, token)
}

// Generate creates a new JWT token
func (s *JWTTokenStore) Generate(ctx context.Context, claims jwtauth.JWTClaims, tokenType jwtauth.TokenType, ttl time.Duration) (string, error) {
	return s.hs256.Generate(ctx, claims, tokenType, ttl)
}

// Revoke invalidates a refresh token by its jti claim
func (s *JWTTokenStore) Revoke(_ context.Context, claims map[string]any) error {
	jti := getJTI(claims)
	if jti != "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.revoked[jti] = true
	}
	return nil
}

// IsRevoked checks if a refresh token has been revoked
func (s *JWTTokenStore) IsRevoked(_ context.Context, claims map[string]any) (bool, error) {
	jti := getJTI(claims)
	if jti == "" {
		return false, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revoked[jti], nil
}

// Close releases resources associated with the store.
// For JWTTokenStore, this is a no-op.
func (s *JWTTokenStore) Close() error {
	return nil
}

// getJTI extracts the jti claim from claims
func getJTI(claims map[string]any) string {
	if jti, ok := claims["jti"].(string); ok {
		return jti
	}
	return ""
}

func main() {
	app := zh.New()

	// HS256 configuration using zero-dependency built-in implementation
	hp := jwtauth.HS256Config{
		Secret: refreshJWTSecret,
		Issuer: "zerohttp-example",
	}

	// Create TokenStore with HS256 and revocation support
	tokenStore := NewJWTTokenStore(refreshJWTSecret, hp)

	jwtCfg := jwtauth.Config{
		Store:           tokenStore,
		RequiredClaims:  []string{"sub"},
		ExcludedPaths:   []string{"/login", "/register"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Public endpoints (no auth required)
	app.POST("/login", refreshLoginHandler(jwtCfg))
	app.POST("/logout", jwtauth.LogoutTokenHandler(jwtCfg)) // Revokes refresh token
	app.POST("/register", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "registration endpoint - implement your logic here",
		})
	}))

	// Refresh token endpoint
	app.POST("/auth/refresh", jwtauth.RefreshTokenHandler(jwtCfg))

	// Protected endpoints (JWT required)
	app.Use(jwtauth.New(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(refreshProfileHandler))
	app.GET("/api/admin", zh.HandlerFunc(refreshAdminHandler))

	fmt.Println("Server starting on http://localhost:8080")
	log.Fatal(app.Start())
}

// refreshLoginHandler authenticates users and returns JWT tokens
func refreshLoginHandler(cfg jwtauth.Config) zh.HandlerFunc {
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

		// Generate tokens
		accessToken, err := jwtauth.GenerateAccessToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate token"})
		}

		refreshToken, err := jwtauth.GenerateRefreshToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate token"})
		}

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    int(cfg.AccessTokenTTL.Seconds()),
		})
	}
}

func refreshProfileHandler(w http.ResponseWriter, r *http.Request) error {
	jwt := jwtauth.GetClaims(r)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": jwt.Subject(),
		"scopes":  jwt.Scopes(),
		"message": "This is your profile",
	})
}

func refreshAdminHandler(w http.ResponseWriter, r *http.Request) error {
	// Check for admin scope
	if !jwtauth.GetClaims(r).HasScope("admin") {
		return zh.R.JSON(w, http.StatusForbidden, zh.M{"error": "admin scope required"})
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message": "Admin access granted",
	})
}

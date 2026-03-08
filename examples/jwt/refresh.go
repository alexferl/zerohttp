//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

var refreshJWTSecret = []byte("your-secret-key-change-in-production")

// JWTTokenStore implements config.TokenStore with HS256 and in-memory revocation.
// In production, use Redis or database for revocation storage.
type JWTTokenStore struct {
	mu      sync.RWMutex
	revoked map[string]bool // map of jti -> revoked
	hs256   *middleware.HS256TokenStore
	opts    middleware.HS256Options
}

// NewJWTTokenStore creates a new TokenStore with HS256 and revocation support
func NewJWTTokenStore(secret []byte, opts middleware.HS256Options) *JWTTokenStore {
	return &JWTTokenStore{
		revoked: make(map[string]bool),
		hs256:   middleware.NewHS256TokenStore(secret, opts),
		opts:    opts,
	}
}

// Validate parses and validates a JWT token
func (s *JWTTokenStore) Validate(token string) (config.JWTClaims, error) {
	return s.hs256.Validate(token)
}

// Generate creates a new JWT token
func (s *JWTTokenStore) Generate(claims config.JWTClaims, tokenType config.TokenType) (string, error) {
	return s.hs256.Generate(claims, tokenType)
}

// Revoke invalidates a refresh token by its jti claim
func (s *JWTTokenStore) Revoke(claims config.JWTClaims) error {
	jti := getJTI(claims)
	if jti != "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.revoked[jti] = true
	}
	return nil
}

// IsRevoked checks if a refresh token has been revoked
func (s *JWTTokenStore) IsRevoked(claims config.JWTClaims) bool {
	jti := getJTI(claims)
	if jti == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revoked[jti]
}

// getJTI extracts the jti claim from claims
func getJTI(claims config.JWTClaims) string {
	switch c := claims.(type) {
	case map[string]any:
		if jti, ok := c["jti"].(string); ok {
			return jti
		}
	case middleware.HS256Claims:
		if jti, ok := c["jti"].(string); ok {
			return jti
		}
	}
	return ""
}

func main() {
	app := zh.New()

	// HS256 configuration using zero-dependency built-in implementation
	hp := middleware.HS256Options{
		Secret: refreshJWTSecret,
		Issuer: "zerohttp-example",
	}

	// Create TokenStore with HS256 and revocation support
	tokenStore := NewJWTTokenStore(refreshJWTSecret, hp)

	jwtCfg := config.JWTAuthConfig{
		TokenStore:      tokenStore,
		RequiredClaims:  []string{"sub"},
		ExemptPaths:     []string{"/login", "/register"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Public endpoints (no auth required)
	app.POST("/login", refreshLoginHandler(jwtCfg))
	app.POST("/logout", middleware.LogoutTokenHandler(jwtCfg)) // Revokes refresh token
	app.POST("/register", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "registration endpoint - implement your logic here",
		})
	}))

	// Refresh token endpoint
	app.POST("/auth/refresh", middleware.RefreshTokenHandler(jwtCfg))

	// Protected endpoints (JWT required)
	app.Use(middleware.JWTAuth(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(refreshProfileHandler))
	app.GET("/api/admin", zh.HandlerFunc(refreshAdminHandler))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  POST /login           - Get access and refresh tokens")
	fmt.Println("  POST /logout          - Revoke refresh token")
	fmt.Println("  POST /register        - Register a new user (stub)")
	fmt.Println("  POST /auth/refresh    - Refresh tokens (checks revocation)")
	fmt.Println("  GET  /api/profile     - Get user profile (requires auth)")
	fmt.Println("  GET  /api/admin       - Admin endpoint (requires auth + admin scope)")
	fmt.Println()
	fmt.Println("Try:")
	fmt.Println("  # Login and get tokens")
	fmt.Println("  curl -X POST http://localhost:8080/login -d '{\"username\":\"alice\",\"password\":\"secret\"}'")
	fmt.Println()
	fmt.Println("  # Access protected endpoint")
	fmt.Println("  curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/profile")
	fmt.Println()
	fmt.Println("  # Refresh tokens (will fail if revoked)")
	fmt.Println("  curl -X POST http://localhost:8080/auth/refresh -d '{\"refresh_token\":\"<refresh_token>\"}'")
	fmt.Println()
	fmt.Println("  # Logout (revokes refresh token)")
	fmt.Println("  curl -X POST http://localhost:8080/logout -d '{\"refresh_token\":\"<refresh_token>\"}'")
	fmt.Println()
	log.Fatal(app.Start())
}

// refreshLoginHandler authenticates users and returns JWT tokens
func refreshLoginHandler(cfg config.JWTAuthConfig) zh.HandlerFunc {
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
		claims := middleware.HS256Claims{
			"sub": req.Username,
			"jti": fmt.Sprintf("%s-%d", req.Username, time.Now().Unix()),
		}

		// Generate tokens
		accessToken, err := middleware.GenerateAccessToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate token"})
		}

		refreshToken, err := middleware.GenerateRefreshToken(r, claims, cfg)
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
	jwt := middleware.GetJWTClaims(r)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": jwt.Subject(),
		"scopes":  jwt.Scopes(),
		"message": "This is your profile",
	})
}

func refreshAdminHandler(w http.ResponseWriter, r *http.Request) error {
	// Check for admin scope
	if !middleware.GetJWTClaims(r).HasScope("admin") {
		return zh.R.JSON(w, http.StatusForbidden, zh.M{"error": "admin scope required"})
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message": "Admin access granted",
	})
}

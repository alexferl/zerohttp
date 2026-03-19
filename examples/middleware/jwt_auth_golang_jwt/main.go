package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	jwt "github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

// GolangJWTTokenStore implements config.TokenStore using github.com/golang-jwt/jwt
type GolangJWTTokenStore struct {
	secret   []byte
	mu       sync.RWMutex
	revoked  map[string]bool // map of jti/exp -> revoked
	sessions map[string]bool // map of sid -> revoked (revokes all tokens in session)
}

// NewGolangJWTTokenStore creates a new TokenStore using golang-jwt/jwt
func NewGolangJWTTokenStore(secret []byte) *GolangJWTTokenStore {
	return &GolangJWTTokenStore{
		secret:   secret,
		revoked:  make(map[string]bool),
		sessions: make(map[string]bool),
	}
}

// Validate parses and validates a JWT token
func (s *GolangJWTTokenStore) Validate(ctx context.Context, tokenString string) (config.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.secret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return token.Claims, nil
}

// Generate creates a new JWT token for the given claims
func (s *GolangJWTTokenStore) Generate(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
	mapClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		// Convert map[string]any to jwt.MapClaims
		if m, ok := claims.(map[string]any); ok {
			mapClaims = jwt.MapClaims(m)
		} else {
			return "", fmt.Errorf("unsupported claims type")
		}
	}

	// exp is already set by middleware.GenerateAccessToken/GenerateRefreshToken
	// Just add type claim for refresh tokens
	if tokenType == config.RefreshToken {
		mapClaims["type"] = config.TokenTypeRefresh
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mapClaims)
	return token.SignedString(s.secret)
}

// Revoke marks a token and its session as revoked
func (s *GolangJWTTokenStore) Revoke(ctx context.Context, claims map[string]any) error {
	// Revoke by exp+sub (individual token)
	sub, _ := claims["sub"].(string)
	if exp, ok := claims["exp"].(int64); ok {
		key := fmt.Sprintf("%s:%d", sub, exp)
		s.mu.Lock()
		s.revoked[key] = true
		s.mu.Unlock()
	}

	// Revoke entire session (revokes all tokens with same sid)
	if sid, ok := claims["sid"].(string); ok && sid != "" {
		s.mu.Lock()
		s.sessions[sid] = true
		s.mu.Unlock()
	}

	return nil
}

// IsRevoked checks if a token has been revoked
func (s *GolangJWTTokenStore) IsRevoked(ctx context.Context, claims map[string]any) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if session is revoked
	if sid, ok := claims["sid"].(string); ok && sid != "" {
		if s.sessions[sid] {
			return true, nil
		}
	}

	// Check if individual token is revoked
	sub, _ := claims["sub"].(string)
	if exp, ok := claims["exp"].(int64); ok {
		key := fmt.Sprintf("%s:%d", sub, exp)
		if s.revoked[key] {
			return true, nil
		}
	}

	return false, nil
}

func main() {
	app := zh.New()

	// Create TokenStore using golang-jwt/jwt
	tokenStore := NewGolangJWTTokenStore(jwtSecret)

	jwtCfg := config.JWTAuthConfig{
		TokenStore:      tokenStore,
		RequiredClaims:  []string{"sub"},
		ExcludedPaths:   []string{"/login"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Public login endpoint
	app.POST("/login", loginHandlerGolangJWT(jwtCfg))

	// Refresh token endpoint
	app.POST("/auth/refresh", middleware.RefreshTokenHandler(jwtCfg))

	// Protected endpoints
	app.Use(middleware.JWTAuth(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(profileHandlerGolangJWT))

	log.Fatal(app.Start())
}

func loginHandlerGolangJWT(cfg config.JWTAuthConfig) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := zh.B.JSON(r.Body, &req); err != nil {
			return zh.R.JSON(w, http.StatusBadRequest, zh.M{"error": "invalid request"})
		}

		// Demo credentials
		if req.Username != "alice" || req.Password != "secret" {
			return zh.R.JSON(w, http.StatusUnauthorized, zh.M{"error": "invalid credentials"})
		}

		// Generate a session ID that links access and refresh tokens
		sessionID := fmt.Sprintf("%s_%d", req.Username, time.Now().UnixNano())

		claims := map[string]any{
			"sub":   req.Username,
			"scope": "read write",
			"sid":   sessionID,
		}

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

func profileHandlerGolangJWT(w http.ResponseWriter, r *http.Request) error {
	jwtWrapper := middleware.GetJWTClaims(r)

	// Type assert to map[string]any (claims are normalized by middleware)
	claims := jwtWrapper.Raw()
	if claims == nil {
		return zh.R.JSON(w, http.StatusUnauthorized, zh.M{"error": "no claims found"})
	}

	m, ok := claims.(map[string]any)
	if !ok {
		return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "invalid claims type"})
	}

	subject, _ := m["sub"].(string)
	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": subject,
		"message": "Hello from golang-jwt/jwt example",
		"scopes":  jwtWrapper.Scopes(),
	})
}

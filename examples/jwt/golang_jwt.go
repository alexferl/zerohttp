//go:build ignore
// +build ignore

// This example demonstrates JWT authentication using github.com/golang-jwt/jwt
//
// To run this example:
//   go get github.com/golang-jwt/jwt/v5
//   go run golang_jwt.go
//
// Test commands:
//   # 1. Login to get tokens
//   curl -X POST http://localhost:8080/login \
//     -H "Content-Type: application/json" \
//     -d '{"username":"alice","password":"secret"}'
//
//   # 2. Access protected endpoint (replace <token> with access_token from step 1)
//   curl -H "Authorization: Bearer <token>" http://localhost:8080/api/profile
//
//   # 3. Refresh tokens (replace <refresh_token> from step 1)
//   curl -X POST http://localhost:8080/auth/refresh \
//     -H "Content-Type: application/json" \
//     -d '{"refresh_token":"<refresh_token>"}'

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

// GolangJWTTokenStore implements config.TokenStore using github.com/golang-jwt/jwt
type GolangJWTTokenStore struct {
	secret []byte
}

// NewGolangJWTTokenStore creates a new TokenStore using golang-jwt/jwt
func NewGolangJWTTokenStore(secret []byte) *GolangJWTTokenStore {
	return &GolangJWTTokenStore{secret: secret}
}

// Validate parses and validates a JWT token
func (s *GolangJWTTokenStore) Validate(tokenString string) (config.JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
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
func (s *GolangJWTTokenStore) Generate(claims config.JWTClaims, tokenType config.TokenType) (string, error) {
	mapClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		// Convert map[string]any to jwt.MapClaims
		if m, ok := claims.(map[string]any); ok {
			mapClaims = jwt.MapClaims(m)
		} else {
			return "", fmt.Errorf("unsupported claims type")
		}
	}

	// Set expiration based on token type
	ttl := config.DefaultJWTAuthConfig.AccessTokenTTL
	if tokenType == config.RefreshToken {
		ttl = config.DefaultJWTAuthConfig.RefreshTokenTTL
		mapClaims["type"] = config.TokenTypeRefresh
	}
	mapClaims["exp"] = time.Now().Add(ttl).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mapClaims)
	return token.SignedString(s.secret)
}

// Revoke is a no-op in this example. In production, store revoked jti in Redis/DB.
func (s *GolangJWTTokenStore) Revoke(claims config.JWTClaims) error {
	// No-op for this example
	return nil
}

// IsRevoked always returns false in this example. In production, check Redis/DB.
func (s *GolangJWTTokenStore) IsRevoked(claims config.JWTClaims) bool {
	// Always returns false for this example
	return false
}

func main() {
	app := zh.New()

	// Create TokenStore using golang-jwt/jwt
	tokenStore := NewGolangJWTTokenStore(jwtSecret)

	jwtCfg := config.JWTAuthConfig{
		TokenStore:      tokenStore,
		RequiredClaims:  []string{"sub"},
		ExemptPaths:     []string{"/login"},
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

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()
	fmt.Println("This example uses github.com/golang-jwt/jwt v5")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  POST /login        - Get tokens")
	fmt.Println("  POST /auth/refresh - Refresh tokens")
	fmt.Println("  GET  /api/profile  - Get profile (requires auth)")
	fmt.Println()
	fmt.Println("Try these commands:")
	fmt.Println("  # 1. Login to get tokens")
	fmt.Println(`  curl -X POST http://localhost:8080/login -H "Content-Type: application/json" -d '{"username":"alice","password":"secret"}'`)
	fmt.Println()
	fmt.Println("  # 2. Access protected endpoint (replace <token> with access_token)")
	fmt.Println(`  curl -H "Authorization: Bearer <token>" http://localhost:8080/api/profile`)
	fmt.Println()
	fmt.Println("  # 3. Refresh tokens (replace <refresh_token>)")
	fmt.Println(`  curl -X POST http://localhost:8080/auth/refresh -H "Content-Type: application/json" -d '{"refresh_token":"<refresh_token>"}'`)
	fmt.Println()
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

		claims := map[string]any{
			"sub":   req.Username,
			"scope": "read write",
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

	// Type assert to jwt.Claims
	claims := jwtWrapper.Raw()
	if claims == nil {
		return zh.R.JSON(w, http.StatusUnauthorized, zh.M{"error": "no claims found"})
	}

	jwtClaims, ok := claims.(jwt.Claims)
	if !ok {
		return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "invalid claims type"})
	}

	subject, _ := jwtClaims.GetSubject()

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": subject,
		"message": "Hello from golang-jwt/jwt example",
		"scopes":  jwtWrapper.Scopes(),
	})
}

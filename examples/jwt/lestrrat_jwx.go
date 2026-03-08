//go:build ignore
// +build ignore

// This example demonstrates JWT authentication using github.com/lestrrat-go/jwx v3
//
// To run this example:
//   go get github.com/lestrrat-go/jwx/v3/jwt
//   go get github.com/lestrrat-go/jwx/v3/jwk
//   go run lestrrat_jwx.go
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

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

// LestrratTokenStore implements config.TokenStore using github.com/lestrrat-go/jwx
type LestrratTokenStore struct {
	keySet jwk.Set
}

// NewLestrratTokenStore creates a new TokenStore using lestrrat-go/jwx
func NewLestrratTokenStore(keySet jwk.Set) *LestrratTokenStore {
	return &LestrratTokenStore{keySet: keySet}
}

// Validate parses and validates a JWT token
func (s *LestrratTokenStore) Validate(tokenString string) (config.JWTClaims, error) {
	token, err := jwt.ParseString(tokenString,
		jwt.WithKeySet(s.keySet),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// Generate creates a new JWT token for the given claims
func (s *LestrratTokenStore) Generate(claims config.JWTClaims, tokenType config.TokenType) (string, error) {
	// Build JWT token
	builder := jwt.NewBuilder()

	// Add claims from map
	if m, ok := claims.(map[string]any); ok {
		for k, v := range m {
			builder.Claim(k, v)
		}
	}

	// Set expiration based on token type
	ttl := config.DefaultJWTAuthConfig.AccessTokenTTL
	if tokenType == config.RefreshToken {
		ttl = config.DefaultJWTAuthConfig.RefreshTokenTTL
		builder.Claim("type", config.TokenTypeRefresh)
	}
	builder.Expiration(time.Now().Add(ttl))

	token, err := builder.Build()
	if err != nil {
		return "", err
	}

	// Get the signing key from the keyset
	key, _ := s.keySet.Key(0)

	// Sign the token
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, key))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

// Revoke is a no-op in this example. In production, store revoked jti in Redis/DB.
func (s *LestrratTokenStore) Revoke(claims config.JWTClaims) error {
	// No-op for this example
	return nil
}

// IsRevoked always returns false in this example. In production, check Redis/DB.
func (s *LestrratTokenStore) IsRevoked(claims config.JWTClaims) bool {
	// Always returns false for this example
	return false
}

func main() {
	app := zh.New()

	// Create a symmetric key for HS256
	// In production, load this from a secure location
	key, err := jwk.GenerateKey(jwa.HS256)
	if err != nil {
		log.Fatalf("failed to generate key: %s", err)
	}
	keySet := jwk.NewSet()
	keySet.AddKey(key)

	// Create TokenStore using lestrrat-go/jwx
	tokenStore := NewLestrratTokenStore(keySet)

	jwtCfg := config.JWTAuthConfig{
		TokenStore:      tokenStore,
		RequiredClaims:  []string{"sub"},
		ExemptPaths:     []string{"/login"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Public login endpoint
	app.POST("/login", loginHandlerLestrrat(jwtCfg, key))

	// Refresh token endpoint
	app.POST("/auth/refresh", middleware.RefreshTokenHandler(jwtCfg))

	// Protected endpoints
	app.Use(middleware.JWTAuth(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(profileHandlerLestrrat))

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println()
	fmt.Println("This example uses github.com/lestrrat-go/jwx v3")
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

func loginHandlerLestrrat(cfg config.JWTAuthConfig, key jwk.Key) zh.HandlerFunc {
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

func profileHandlerLestrrat(w http.ResponseWriter, r *http.Request) error {
	jwtWrapper := middleware.GetJWTClaims(r)

	// Type assert to jwt.Token
	claims := jwtWrapper.Raw()
	if claims == nil {
		return zh.R.JSON(w, http.StatusUnauthorized, zh.M{"error": "no claims found"})
	}

	token, ok := claims.(jwt.Token)
	if !ok {
		return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "invalid claims type"})
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": token.Subject(),
		"message": "Hello from lestrrat-go/jwx v3 example",
		"scopes":  jwtWrapper.Scopes(),
	})
}

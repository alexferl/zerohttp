package main

import (
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

func main() {
	app := zh.New()

	// Simple HS256 token store (no refresh tokens, just access tokens)
	hp := middleware.HS256Options{
		Secret: jwtSecret,
		Issuer: "zerohttp-example",
	}

	jwtCfg := config.JWTAuthConfig{
		TokenStore:     middleware.NewHS256TokenStore(jwtSecret, hp),
		RequiredClaims: []string{"sub"},
		ExemptPaths:    []string{"/login", "/register"},
		// Long expiry - 30 days (or use 0 for no expiry)
		AccessTokenTTL: 30 * 24 * time.Hour,
	}

	// Public endpoints (no auth required)
	app.POST("/login", loginHandler(jwtCfg))
	app.POST("/register", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message": "registration endpoint - implement your logic here",
		})
	}))

	// Protected endpoints (JWT required)
	app.Use(middleware.JWTAuth(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(profileHandler))
	app.GET("/api/admin", zh.HandlerFunc(adminHandler))

	log.Fatal(app.Start())
}

// loginHandler authenticates users and returns a JWT token
func loginHandler(cfg config.JWTAuthConfig) zh.HandlerFunc {
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

		// Create claims
		claims := middleware.HS256Claims{
			"sub":   req.Username,
			"scope": "read write",
		}

		// Generate token
		token, err := middleware.GenerateAccessToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate token"})
		}

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   int(cfg.AccessTokenTTL.Seconds()),
		})
	}
}

func profileHandler(w http.ResponseWriter, r *http.Request) error {
	jwt := middleware.GetJWTClaims(r)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": jwt.Subject(),
		"scopes":  jwt.Scopes(),
		"message": "This is your profile",
	})
}

func adminHandler(w http.ResponseWriter, r *http.Request) error {
	// Check for admin scope
	if !middleware.GetJWTClaims(r).HasScope("admin") {
		return zh.R.JSON(w, http.StatusForbidden, zh.M{"error": "admin scope required"})
	}

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"message": "Admin access granted",
	})
}

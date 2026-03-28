package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"

	zh "github.com/alexferl/zerohttp"
)

// LestrratTokenStore implements config.TokenStore using github.com/lestrrat-go/jwx
type LestrratTokenStore struct {
	keySet   jwk.Set
	mu       sync.RWMutex
	revoked  map[string]bool // map of exp:sub -> revoked
	sessions map[string]bool // map of sid -> revoked
}

// NewLestrratTokenStore creates a new TokenStore using lestrrat-go/jwx
func NewLestrratTokenStore(keySet jwk.Set) *LestrratTokenStore {
	return &LestrratTokenStore{
		keySet:   keySet,
		revoked:  make(map[string]bool),
		sessions: make(map[string]bool),
	}
}

// Validate parses and validates a JWT token
func (s *LestrratTokenStore) Validate(ctx context.Context, tokenString string) (jwtauth.JWTClaims, error) {
	// Get the key directly from the keyset (we only have one key)
	key, _ := s.keySet.Key(0)

	token, err := jwt.ParseString(tokenString,
		jwt.WithKey(jwa.HS256(), key),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, err
	}

	// Convert jwt.Token to map[string]any for middleware compatibility
	claims := tokenToMap(token)
	return claims, nil
}

// tokenToMap converts a jwt.Token to map[string]any
func tokenToMap(token jwt.Token) map[string]any {
	m := make(map[string]any)

	// Copy standard claims
	if sub, ok := token.Subject(); ok {
		m["sub"] = sub
	}
	if iss, ok := token.Issuer(); ok {
		m["iss"] = iss
	}
	if aud, ok := token.Audience(); ok {
		m["aud"] = aud
	}
	if exp, ok := token.Expiration(); ok {
		m["exp"] = exp.Unix()
	}
	if iat, ok := token.IssuedAt(); ok {
		m["iat"] = iat.Unix()
	}
	if nbf, ok := token.NotBefore(); ok {
		m["nbf"] = nbf.Unix()
	}
	if jti, ok := token.JwtID(); ok {
		m["jti"] = jti
	}

	// Copy custom claims we care about
	var sid string
	if err := token.Get("sid", &sid); err == nil {
		m["sid"] = sid
	}
	var scope string
	if err := token.Get("scope", &scope); err == nil {
		m["scope"] = scope
	}
	var typ string
	if err := token.Get("type", &typ); err == nil {
		m["type"] = typ
	}

	return m
}

// Generate creates a new JWT token for the given claims
func (s *LestrratTokenStore) Generate(_ context.Context, claims jwtauth.JWTClaims, tokenType jwtauth.TokenType, ttl time.Duration) (string, error) {
	// Build JWT token
	builder := jwt.NewBuilder()

	// Add claims from map
	if m, ok := claims.(map[string]any); ok {
		for k, v := range m {
			switch k {
			// lestrrat-go/jwx has specific methods for standard claims
			case "sub":
				if s, ok := v.(string); ok {
					builder.Subject(s)
				}
			case "iss":
				if s, ok := v.(string); ok {
					builder.Issuer(s)
				}
			case "aud":
				if s, ok := v.(string); ok {
					builder.Audience([]string{s})
				} else if a, ok := v.([]string); ok {
					builder.Audience(a)
				}
			case "iat":
				if i, ok := v.(int64); ok {
					builder.IssuedAt(time.Unix(i, 0))
				} else if f, ok := v.(float64); ok {
					builder.IssuedAt(time.Unix(int64(f), 0))
				}
			case "nbf":
				if i, ok := v.(int64); ok {
					builder.NotBefore(time.Unix(i, 0))
				} else if f, ok := v.(float64); ok {
					builder.NotBefore(time.Unix(int64(f), 0))
				}
			case "exp":
				switch expVal := v.(type) {
				case int64:
					builder.Expiration(time.Unix(expVal, 0))
				case int:
					builder.Expiration(time.Unix(int64(expVal), 0))
				case float64:
					builder.Expiration(time.Unix(int64(expVal), 0))
				}
			case "jti":
				if s, ok := v.(string); ok {
					builder.JwtID(s)
				}
			default:
				builder.Claim(k, v)
			}
		}
	}

	// Add type claim for refresh tokens
	if tokenType == jwtauth.RefreshToken {
		builder.Claim("type", jwtauth.TokenTypeRefresh)
	}

	token, err := builder.Build()
	if err != nil {
		return "", err
	}

	// Get the signing key from the keyset
	key, _ := s.keySet.Key(0)

	// Sign the token
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), key))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

// Revoke marks a token and its session as revoked
func (s *LestrratTokenStore) Revoke(_ context.Context, claims map[string]any) error {
	// Revoke by exp+sub (individual token)
	sub, _ := claims["sub"].(string)
	if exp, ok := claims["exp"].(int64); ok {
		key := fmt.Sprintf("%s:%d", sub, exp)
		s.mu.Lock()
		s.revoked[key] = true
		s.mu.Unlock()
	}

	// Revoke entire session
	if sid, ok := claims["sid"].(string); ok && sid != "" {
		s.mu.Lock()
		s.sessions[sid] = true
		s.mu.Unlock()
	}

	return nil
}

// IsRevoked checks if a token has been revoked
func (s *LestrratTokenStore) IsRevoked(_ context.Context, claims map[string]any) (bool, error) {
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

// Close releases resources associated with the store.
// For LestrratTokenStore, this is a no-op.
func (s *LestrratTokenStore) Close() error {
	return nil
}

func main() {
	app := zh.New()

	// Create a symmetric key for HS256
	// In production, load this from a secure location
	rawKey := []byte("your-secret-key-at-least-32-bytes-long!")
	key, err := jwk.Import(rawKey)
	if err != nil {
		log.Fatalf("failed to import key: %s", err)
	}
	keySet := jwk.NewSet()
	keySet.AddKey(key)

	// Create TokenStore using lestrrat-go/jwx
	tokenStore := NewLestrratTokenStore(keySet)

	jwtCfg := jwtauth.Config{
		Store:           tokenStore,
		RequiredClaims:  []string{"sub"},
		ExcludedPaths:   []string{"/login"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Public login endpoint
	app.POST("/login", loginHandlerLestrrat(jwtCfg, key))

	// Refresh token endpoint
	app.POST("/auth/refresh", jwtauth.RefreshTokenHandler(jwtCfg))

	// Protected endpoints
	app.Use(jwtauth.New(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(profileHandlerLestrrat))

	log.Fatal(app.Start())
}

func loginHandlerLestrrat(cfg jwtauth.Config, _ jwk.Key) zh.HandlerFunc {
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

func profileHandlerLestrrat(w http.ResponseWriter, r *http.Request) error {
	jwtWrapper := jwtauth.GetClaims(r)

	// Type assert to map (Validate now returns map[string]any)
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
		"message": "Hello from lestrrat-go/jwx v3 example",
		"scopes":  jwtWrapper.Scopes(),
	})
}

package jwtauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HS256Store implements the TokenStore interface using HMAC-SHA256.
// This provides a zero-dependency JWT implementation using only the standard library.
type HS256Store struct {
	secret []byte
	opts   HS256Config
}

// NewHS256Store creates a new HS256Store.
// This provides a zero-dependency JWT implementation that satisfies the Store interface.
//
// Example:
//
//	store := jwtauth.NewHS256Store([]byte("your-secret"), jwtauth.HS256Config{
//	    Issuer: "my-app",
//	})
//
//	cfg := Config{
//	    Store: store,
//	}
func NewHS256Store(secret []byte, opts HS256Config) *HS256Store {
	if len(secret) < 32 {
		panic(fmt.Sprintf("HS256 secret must be at least 32 bytes, got %d bytes. Use a cryptographically secure random key.", len(secret)))
	}
	return &HS256Store{
		secret: secret,
		opts:   opts,
	}
}

// Validate parses and validates an HS256 JWT token.
func (s *HS256Store) Validate(_ context.Context, token string) (JWTClaims, error) {
	return parseHS256Token(token, s.secret, s.opts)
}

// Generate creates a new HS256 JWT token for the given claims.
func (s *HS256Store) Generate(_ context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
	return generateHS256Token(claims, s.secret, s.opts)
}

// Revoke is a no-op for HS256Store. In-memory revocation is not supported.
// Use a database-backed Store implementation for revocation support.
func (s *HS256Store) Revoke(_ context.Context, claims map[string]any) error {
	// No-op: HS256TokenStore doesn't support revocation
	// Users should implement their own TokenStore with Redis/DB for revocation
	return nil
}

// IsRevoked always returns (false, nil) for HS256Store.
// Use a database-backed Store implementation for revocation support.
func (s *HS256Store) IsRevoked(_ context.Context, claims map[string]any) (bool, error) {
	// Always returns false: HS256TokenStore doesn't support revocation
	// Users should implement their own TokenStore with Redis/DB for revocation
	return false, nil
}

// HS256Config configures the built-in HS256 JWT implementation.
//
// Security Note: This implementation uses HMAC-SHA256 symmetric signing. It is suitable
// for simple use cases and when you control both token issuance and validation. For
// production systems requiring asymmetric keys (RS256, ES256, EdDSA), key rotation,
// or JWKS support, use a proper JWT library like golang-jwt/jwt or lestrrat-go/jwx.
//
// The Secret must be kept secure. Use a cryptographically secure random key with
// at least 256 bits (32 bytes) of entropy. Do not hardcode secrets in source code.
type HS256Config struct {
	// Secret is the HMAC secret key. Must be at least 32 bytes for security.
	Secret []byte

	// Issuer is the JWT issuer (iss claim)
	Issuer string

	// Audience is the JWT audience (aud claim)
	Audience string

	// ValidateIssuer validates the issuer claim
	ValidateIssuer bool

	// ValidateAudience validates the audience claim
	ValidateAudience bool
}

// HS256Claims represents JWT claims for the built-in HS256 implementation
type HS256Claims map[string]any

// HS256Validator creates a TokenValidator function for HS256 tokens
// This provides a zero-dependency JWT implementation using only the standard library
func HS256Validator(secret []byte, opts HS256Config) func(token string) (JWTClaims, error) {
	return func(token string) (JWTClaims, error) {
		return parseHS256Token(token, secret, opts)
	}
}

// HS256Generator creates a TokenGenerator function for HS256 tokens
func HS256Generator(secret []byte, opts HS256Config) func(claims JWTClaims, tokenType TokenType) (string, error) {
	return func(claims JWTClaims, tokenType TokenType) (string, error) {
		return generateHS256Token(claims, secret, opts)
	}
}

// GetHS256Subject extracts the subject from HS256 claims
func GetHS256Subject(claims JWTClaims) string {
	hsClaims, ok := claims.(HS256Claims)
	if !ok {
		return ""
	}

	if sub, ok := hsClaims["sub"].(string); ok {
		return sub
	}
	return ""
}

// GetHS256Expiration extracts the expiration time from HS256 claims
func GetHS256Expiration(claims JWTClaims) time.Time {
	hsClaims, ok := claims.(HS256Claims)
	if !ok {
		return time.Time{}
	}

	if exp, ok := hsClaims["exp"].(float64); ok {
		return time.Unix(int64(exp), 0)
	}
	return time.Time{}
}

// parseHS256Token parses and validates an HS256 JWT token
func parseHS256Token(tokenString string, secret []byte, opts HS256Config) (JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerB64, payloadB64, signatureB64 := parts[0], parts[1], parts[2]

	headerJSON, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return nil, fmt.Errorf("invalid header: %w", err)
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("invalid header JSON: %w", err)
	}

	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	var claims HS256Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}

	expectedSignature := signHS256(headerB64+"."+payloadB64, secret)
	if signatureB64 != expectedSignature {
		return nil, errors.New("invalid signature")
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, errors.New("token expired")
		}
	}

	if nbf, ok := claims["nbf"].(float64); ok {
		if time.Now().Unix() < int64(nbf) {
			return nil, errors.New("token not yet valid")
		}
	}

	if opts.ValidateIssuer && opts.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != opts.Issuer {
			return nil, errors.New("invalid issuer")
		}
	}

	if opts.ValidateAudience && opts.Audience != "" {
		aud, ok := claims["aud"]
		if !ok {
			return nil, errors.New("missing audience")
		}

		switch v := aud.(type) {
		case string:
			if v != opts.Audience {
				return nil, errors.New("invalid audience")
			}
		case []any:
			found := false
			for _, a := range v {
				if s, ok := a.(string); ok && s == opts.Audience {
					found = true
					break
				}
			}
			if !found {
				return nil, errors.New("invalid audience")
			}
		default:
			return nil, errors.New("invalid audience format")
		}
	}

	return claims, nil
}

// generateHS256Token generates an HS256 JWT token
func generateHS256Token(claims JWTClaims, secret []byte, opts HS256Config) (string, error) {
	hsClaims, ok := claims.(HS256Claims)
	if !ok {
		switch c := claims.(type) {
		case map[string]any:
			hsClaims = c
		default:
			return "", errors.New("unsupported claims type for HS256")
		}
	}

	if opts.Issuer != "" {
		hsClaims["iss"] = opts.Issuer
	}

	if opts.Audience != "" {
		hsClaims["aud"] = opts.Audience
	}

	if _, ok := hsClaims["iat"]; !ok {
		hsClaims["iat"] = time.Now().Unix()
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	payloadJSON, err := json.Marshal(hsClaims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signature := signHS256(headerB64+"."+payloadB64, secret)

	return headerB64 + "." + payloadB64 + "." + signature, nil
}

// signHS256 creates an HMAC-SHA256 signature
func signHS256(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

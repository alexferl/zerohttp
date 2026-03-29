package jwtauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestHS256Validator_Success(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)
	validator := HS256Validator(secret, opts)

	// Generate a token
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Validate the token
	validatedClaims, err := validator(token)
	zhtest.AssertNoError(t, err)

	hsClaims, ok := validatedClaims.(HS256Claims)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, "user123", hsClaims["sub"])
}

func TestHS256Validator_InvalidSignature(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	wrongSecret := []byte("wrong-secret")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)

	// Generate a token with correct secret
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Create validator with wrong secret
	wrongValidator := HS256Validator(wrongSecret, opts)

	_, err = wrongValidator(token)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "invalid signature")
}

func TestHS256Validator_ExpiredToken(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)
	validator := HS256Validator(secret, opts)

	// Generate an expired token
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(-time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	_, err = validator(token)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "expired")
}

func TestHS256Validator_NotBefore(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)
	validator := HS256Validator(secret, opts)

	// Generate a token that is not yet valid
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
		"nbf": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	_, err = validator(token)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "not yet valid")
}

func TestHS256Validator_InvalidFormat(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Invalid token format
	tests := []string{
		"invalid",
		"only.two.parts",
		"too.many.parts.here.now",
	}

	for _, token := range tests {
		_, err := validator(token)
		zhtest.AssertError(t, err)
	}
}

func TestHS256Validator_InvalidAlgorithm(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Token with unsupported algorithm (this is a manually crafted token header)
	// Header: {"alg":"RS256","typ":"JWT"}
	invalidToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.invalid"

	_, err := validator(invalidToken)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "unsupported algorithm")
}

func TestHS256Validator_ValidateIssuer(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer:         "my-app",
		ValidateIssuer: true,
	}

	generator := HS256Generator(secret, opts)
	validator := HS256Validator(secret, opts)

	// Generate a valid token
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Should validate successfully (generator adds issuer)
	_, err = validator(token)
	zhtest.AssertNoError(t, err)

	// Test with wrong issuer
	wrongOpts := HS256Config{
		Issuer:         "wrong-app",
		ValidateIssuer: true,
	}
	wrongValidator := HS256Validator(secret, wrongOpts)

	_, err = wrongValidator(token)
	zhtest.AssertError(t, err)
}

func TestHS256Validator_ValidateAudience(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Audience:         "my-api",
		ValidateAudience: true,
	}

	generator := HS256Generator(secret, opts)
	validator := HS256Validator(secret, opts)

	// Generate a valid token
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Should validate successfully
	_, err = validator(token)
	zhtest.AssertNoError(t, err)

	// Test with wrong audience
	wrongOpts := HS256Config{
		Audience:         "wrong-api",
		ValidateAudience: true,
	}
	wrongValidator := HS256Validator(secret, wrongOpts)

	_, err = wrongValidator(token)
	zhtest.AssertError(t, err)
}

func TestHS256Generator_WithMapClaims(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer: "test-issuer",
	}

	generator := HS256Generator(secret, opts)

	// Use regular map[string]any instead of HS256Claims
	claims := map[string]any{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Token should have 3 parts
	parts := strings.Split(token, ".")
	zhtest.AssertEqual(t, 3, len(parts))
}

func TestHS256Generator_UnsupportedClaimsType(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)

	// Use unsupported claims type
	_, err := generator("not a map", AccessToken)
	zhtest.AssertError(t, err)
}

func TestGetHS256Subject(t *testing.T) {
	claims := HS256Claims{
		"sub": "user123",
	}

	sub := GetHS256Subject(claims)
	zhtest.AssertEqual(t, "user123", sub)

	// Test with non-HS256Claims
	sub = GetHS256Subject(map[string]any{"sub": "user456"})
	zhtest.AssertEqual(t, "", sub)
}

func TestGetHS256Expiration(t *testing.T) {
	expTime := time.Now().Add(time.Hour)
	claims := HS256Claims{
		"exp": float64(expTime.Unix()),
	}

	got := GetHS256Expiration(claims)
	zhtest.AssertEqual(t, time.Unix(expTime.Unix(), 0), got)

	// Test with non-HS256Claims
	got = GetHS256Expiration(map[string]any{"exp": float64(expTime.Unix())})
	zhtest.AssertTrue(t, got.IsZero())
}

func TestSignHS256(t *testing.T) {
	secret := []byte("my-secret")
	data := "header.payload"

	sig1 := signHS256(data, secret)
	sig2 := signHS256(data, secret)

	// Same input should produce same signature
	zhtest.AssertEqual(t, sig1, sig2)

	// Different secret should produce different signature
	otherSig := signHS256(data, []byte("other-secret"))
	zhtest.AssertNotEqual(t, sig1, otherSig)
}

// Tests for HS256TokenStore (TokenStore interface implementation)

func TestNewHS256TokenStore(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}

	store := NewHS256Store(secret, opts)
	zhtest.AssertNotNil(t, store)
}

func TestNewHS256TokenStore_ShortSecretPanics(t *testing.T) {
	zhtest.AssertPanicContains(t, func() {
		// This should panic - secret is only 13 bytes
		_ = NewHS256Store([]byte("short-secret"), HS256Config{})
	}, "HS256 secret must be at least 32 bytes")
}

func TestHS256TokenStore_Validate(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// Generate a token first
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := store.Generate(context.Background(), claims, AccessToken, 15*time.Minute)
	zhtest.AssertNoError(t, err)

	// Validate the token
	validatedClaims, err := store.Validate(context.Background(), token)
	zhtest.AssertNoError(t, err)

	hsClaims, ok := validatedClaims.(HS256Claims)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, "user123", hsClaims["sub"])
}

func TestHS256TokenStore_Validate_InvalidToken(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	_, err := store.Validate(context.Background(), "invalid.token.here")
	zhtest.AssertError(t, err)
}

func TestHS256TokenStore_Generate(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer: "test-issuer",
	}
	store := NewHS256Store(secret, opts)

	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := store.Generate(context.Background(), claims, AccessToken, 15*time.Minute)
	zhtest.AssertNoError(t, err)

	// Token should have 3 parts
	parts := strings.Split(token, ".")
	zhtest.AssertEqual(t, 3, len(parts))
}

func TestHS256TokenStore_Generate_WithMapClaims(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// Use regular map[string]any
	claims := map[string]any{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token, err := store.Generate(context.Background(), claims, AccessToken, 15*time.Minute)
	zhtest.AssertNoError(t, err)

	// Token should have 3 parts
	parts := strings.Split(token, ".")
	zhtest.AssertEqual(t, 3, len(parts))
}

func TestHS256TokenStore_Generate_UnsupportedClaimsType(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// Use unsupported claims type
	_, err := store.Generate(context.Background(), "not a map", AccessToken, 15*time.Minute)
	zhtest.AssertError(t, err)
}

func TestHS256TokenStore_Revoke(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// Revoke is a no-op for HS256TokenStore
	claims := HS256Claims{"sub": "user123"}
	err := store.Revoke(context.Background(), claims)
	zhtest.AssertNoError(t, err)
}

func TestHS256TokenStore_IsRevoked(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// IsRevoked always returns false for HS256TokenStore
	claims := HS256Claims{"sub": "user123", "jti": "token-id"}
	revoked, _ := store.IsRevoked(context.Background(), claims)
	zhtest.AssertFalse(t, revoked)
}

func TestHS256TokenStore_RoundTrip(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}
	store := NewHS256Store(secret, opts)

	// Generate token
	claims := HS256Claims{
		"sub":   "user123",
		"scope": "read write",
		"exp":   float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := store.Generate(context.Background(), claims, AccessToken, 15*time.Minute)
	zhtest.AssertNoError(t, err)

	// Validate token
	validatedClaims, err := store.Validate(context.Background(), token)
	zhtest.AssertNoError(t, err)

	hsClaims, ok := validatedClaims.(HS256Claims)
	zhtest.AssertTrue(t, ok)

	// Check claims survived round trip
	zhtest.AssertEqual(t, "user123", hsClaims["sub"])
	zhtest.AssertEqual(t, "read write", hsClaims["scope"])

	// Issuer and audience should be added by generator
	zhtest.AssertEqual(t, "test-issuer", hsClaims["iss"])
	zhtest.AssertEqual(t, "test-audience", hsClaims["aud"])
}

func TestParseHS256Token_InvalidBase64(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Invalid base64 in header
	_, err := validator("!!!.eyJzdWIiOiJ1c2VyMTIzIn0.signature")
	zhtest.AssertError(t, err)
}

func TestParseHS256Token_InvalidHeaderJSON(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Valid base64 but invalid JSON
	invalidHeader := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	token := invalidHeader + ".eyJzdWIiOiJ1c2VyMTIzIn0.signature"

	_, err := validator(token)
	zhtest.AssertError(t, err)
}

func TestParseHS256Token_InvalidPayloadBase64(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Valid header, invalid payload base64
	validHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	token := validHeader + ".!!!.signature"

	_, err := validator(token)
	zhtest.AssertError(t, err)
}

func TestParseHS256Token_InvalidPayloadJSON(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Valid header, invalid payload JSON
	validHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	invalidPayload := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	token := validHeader + "." + invalidPayload + ".signature"

	_, err := validator(token)
	zhtest.AssertError(t, err)
}

func TestParseHS256Token_MissingAudience(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Audience:         "my-api",
		ValidateAudience: true,
	}

	validator := HS256Validator(secret, opts)

	// Create a token manually without audience
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user123","exp":` + fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()) + `}`))
	signature := signHS256(header+"."+payload, secret)
	tokenWithoutAud := header + "." + payload + "." + signature

	_, err := validator(tokenWithoutAud)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "audience")
}

func TestParseHS256Token_InvalidAudienceArray(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Audience:         "my-api",
		ValidateAudience: true,
	}

	// Create a token with audience as array that doesn't contain expected audience
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user123","aud":["other-api"],"exp":` + fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()) + `}`))
	signature := signHS256(header+"."+payload, secret)
	token := header + "." + payload + "." + signature

	validator := HS256Validator(secret, opts)
	_, err := validator(token)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "invalid audience")
}

func TestParseHS256Token_AudienceWrongFormat(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Audience:         "my-api",
		ValidateAudience: true,
	}

	// Create a token with audience as number (invalid format)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user123","aud":123,"exp":` + fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()) + `}`))
	signature := signHS256(header+"."+payload, secret)
	token := header + "." + payload + "." + signature

	validator := HS256Validator(secret, opts)
	_, err := validator(token)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "invalid audience format")
}

func TestParseHS256Token_NoExpiration(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)

	// Generate claims without expiration
	claims := HS256Claims{
		"sub": "user123",
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Token without exp should be valid
	validator := HS256Validator(secret, opts)
	validatedClaims, err := validator(token)
	zhtest.AssertNoError(t, err)
	zhtest.AssertNotNil(t, validatedClaims)
}

func TestParseHS256Token_NoNotBefore(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)

	// Generate claims without nbf
	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := generator(claims, AccessToken)
	zhtest.AssertNoError(t, err)

	// Token without nbf should be valid
	validator := HS256Validator(secret, opts)
	validatedClaims, err := validator(token)
	zhtest.AssertNoError(t, err)
	zhtest.AssertNotNil(t, validatedClaims)
}

func TestGenerateHS256Token_NoIssuer(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer: "", // No issuer set
	}

	store := NewHS256Store(secret, opts)

	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := store.Generate(context.Background(), claims, AccessToken, 15*time.Minute)
	zhtest.AssertNoError(t, err)

	// Validate and check issuer wasn't added
	validatedClaims, err := store.Validate(context.Background(), token)
	zhtest.AssertNoError(t, err)

	hsClaims := validatedClaims.(HS256Claims)
	_, ok := hsClaims["iss"]
	zhtest.AssertFalse(t, ok)
}

func TestGenerateHS256Token_NoAudience(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Audience: "", // No audience set
	}

	store := NewHS256Store(secret, opts)

	claims := HS256Claims{
		"sub": "user123",
		"exp": float64(time.Now().Add(time.Hour).Unix()),
	}

	token, err := store.Generate(context.Background(), claims, AccessToken, 15*time.Minute)
	zhtest.AssertNoError(t, err)

	// Validate and check audience wasn't added
	validatedClaims, err := store.Validate(context.Background(), token)
	zhtest.AssertNoError(t, err)

	hsClaims := validatedClaims.(HS256Claims)
	_, ok := hsClaims["aud"]
	zhtest.AssertFalse(t, ok)
}

func TestGetHS256Subject_Missing(t *testing.T) {
	claims := HS256Claims{
		// No "sub" claim
	}

	sub := GetHS256Subject(claims)
	zhtest.AssertEqual(t, "", sub)
}

func TestGetHS256Expiration_Missing(t *testing.T) {
	claims := HS256Claims{
		// No "exp" claim
	}

	exp := GetHS256Expiration(claims)
	zhtest.AssertTrue(t, exp.IsZero())
}

func TestHS256Store_Close(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	err := store.Close()
	zhtest.AssertNoError(t, err)
}

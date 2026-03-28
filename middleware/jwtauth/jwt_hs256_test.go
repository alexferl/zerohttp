package jwtauth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate the token
	validatedClaims, err := validator(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	hsClaims, ok := validatedClaims.(HS256Claims)
	if !ok {
		t.Fatal("expected HS256Claims")
	}

	if hsClaims["sub"] != "user123" {
		t.Errorf("expected sub = 'user123', got %v", hsClaims["sub"])
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Create validator with wrong secret
	wrongValidator := HS256Validator(wrongSecret, opts)

	_, err = wrongValidator(token)
	if err == nil {
		t.Error("expected validation to fail with wrong secret")
	}
	if !strings.Contains(err.Error(), "invalid signature") {
		t.Errorf("expected 'invalid signature' error, got: %v", err)
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = validator(token)
	if err == nil {
		t.Error("expected validation to fail for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected 'expired' error, got: %v", err)
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = validator(token)
	if err == nil {
		t.Error("expected validation to fail for not-yet-valid token")
	}
	if !strings.Contains(err.Error(), "not yet valid") {
		t.Errorf("expected 'not yet valid' error, got: %v", err)
	}
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
		if err == nil {
			t.Errorf("expected error for token %q", token)
		}
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
	if err == nil {
		t.Error("expected error for unsupported algorithm")
	}
	if !strings.Contains(err.Error(), "unsupported algorithm") {
		t.Errorf("expected 'unsupported algorithm' error, got: %v", err)
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Should validate successfully (generator adds issuer)
	_, err = validator(token)
	if err != nil {
		t.Errorf("expected validation to succeed: %v", err)
	}

	// Test with wrong issuer
	wrongOpts := HS256Config{
		Issuer:         "wrong-app",
		ValidateIssuer: true,
	}
	wrongValidator := HS256Validator(secret, wrongOpts)

	_, err = wrongValidator(token)
	if err == nil {
		t.Error("expected validation to fail with wrong issuer")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Should validate successfully
	_, err = validator(token)
	if err != nil {
		t.Errorf("expected validation to succeed: %v", err)
	}

	// Test with wrong audience
	wrongOpts := HS256Config{
		Audience:         "wrong-api",
		ValidateAudience: true,
	}
	wrongValidator := HS256Validator(secret, wrongOpts)

	_, err = wrongValidator(token)
	if err == nil {
		t.Error("expected validation to fail with wrong audience")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token should have 3 parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
}

func TestHS256Generator_UnsupportedClaimsType(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	generator := HS256Generator(secret, opts)

	// Use unsupported claims type
	_, err := generator("not a map", AccessToken)
	if err == nil {
		t.Error("expected error for unsupported claims type")
	}
}

func TestGetHS256Subject(t *testing.T) {
	claims := HS256Claims{
		"sub": "user123",
	}

	sub := GetHS256Subject(claims)
	if sub != "user123" {
		t.Errorf("expected sub = 'user123', got %q", sub)
	}

	// Test with non-HS256Claims
	sub = GetHS256Subject(map[string]any{"sub": "user456"})
	if sub != "" {
		t.Errorf("expected empty string for non-HS256Claims, got %q", sub)
	}
}

func TestGetHS256Expiration(t *testing.T) {
	expTime := time.Now().Add(time.Hour)
	claims := HS256Claims{
		"exp": float64(expTime.Unix()),
	}

	got := GetHS256Expiration(claims)
	if !got.Equal(time.Unix(expTime.Unix(), 0)) {
		t.Errorf("expected exp = %v, got %v", expTime, got)
	}

	// Test with non-HS256Claims
	got = GetHS256Expiration(map[string]any{"exp": float64(expTime.Unix())})
	if !got.IsZero() {
		t.Errorf("expected zero time for non-HS256Claims, got %v", got)
	}
}

func TestSignHS256(t *testing.T) {
	secret := []byte("my-secret")
	data := "header.payload"

	sig1 := signHS256(data, secret)
	sig2 := signHS256(data, secret)

	// Same input should produce same signature
	if sig1 != sig2 {
		t.Error("expected same signature for same input")
	}

	// Different secret should produce different signature
	otherSig := signHS256(data, []byte("other-secret"))
	if sig1 == otherSig {
		t.Error("expected different signature for different secret")
	}
}

// Tests for HS256TokenStore (TokenStore interface implementation)

func TestNewHS256TokenStore(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}

	store := NewHS256Store(secret, opts)
	if store == nil {
		t.Fatal("expected store to be created")
	}
}

func TestNewHS256TokenStore_ShortSecretPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for short secret, but did not panic")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "HS256 secret must be at least 32 bytes") {
			t.Errorf("expected panic message about 32 bytes, got: %v", r)
		}
	}()

	// This should panic - secret is only 13 bytes
	_ = NewHS256Store([]byte("short-secret"), HS256Config{})
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate the token
	validatedClaims, err := store.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	hsClaims, ok := validatedClaims.(HS256Claims)
	if !ok {
		t.Fatal("expected HS256Claims")
	}

	if hsClaims["sub"] != "user123" {
		t.Errorf("expected sub = 'user123', got %v", hsClaims["sub"])
	}
}

func TestHS256TokenStore_Validate_InvalidToken(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	_, err := store.Validate(context.Background(), "invalid.token.here")
	if err == nil {
		t.Error("expected error for invalid token")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token should have 3 parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token should have 3 parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
}

func TestHS256TokenStore_Generate_UnsupportedClaimsType(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// Use unsupported claims type
	_, err := store.Generate(context.Background(), "not a map", AccessToken, 15*time.Minute)
	if err == nil {
		t.Error("expected error for unsupported claims type")
	}
}

func TestHS256TokenStore_Revoke(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// Revoke is a no-op for HS256TokenStore
	claims := HS256Claims{"sub": "user123"}
	err := store.Revoke(context.Background(), claims)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHS256TokenStore_IsRevoked(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	// IsRevoked always returns false for HS256TokenStore
	claims := HS256Claims{"sub": "user123", "jti": "token-id"}
	revoked, _ := store.IsRevoked(context.Background(), claims)
	if revoked {
		t.Error("expected IsRevoked to return false")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate token
	validatedClaims, err := store.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	hsClaims, ok := validatedClaims.(HS256Claims)
	if !ok {
		t.Fatal("expected HS256Claims")
	}

	// Check claims survived round trip
	if hsClaims["sub"] != "user123" {
		t.Errorf("expected sub = 'user123', got %v", hsClaims["sub"])
	}
	if hsClaims["scope"] != "read write" {
		t.Errorf("expected scope = 'read write', got %v", hsClaims["scope"])
	}

	// Issuer and audience should be added by generator
	if hsClaims["iss"] != "test-issuer" {
		t.Errorf("expected iss = 'test-issuer', got %v", hsClaims["iss"])
	}
	if hsClaims["aud"] != "test-audience" {
		t.Errorf("expected aud = 'test-audience', got %v", hsClaims["aud"])
	}
}

func TestParseHS256Token_InvalidBase64(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Invalid base64 in header
	_, err := validator("!!!.eyJzdWIiOiJ1c2VyMTIzIn0.signature")
	if err == nil {
		t.Error("expected error for invalid base64 header")
	}
}

func TestParseHS256Token_InvalidHeaderJSON(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Valid base64 but invalid JSON
	invalidHeader := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	token := invalidHeader + ".eyJzdWIiOiJ1c2VyMTIzIn0.signature"

	_, err := validator(token)
	if err == nil {
		t.Error("expected error for invalid header JSON")
	}
}

func TestParseHS256Token_InvalidPayloadBase64(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}

	validator := HS256Validator(secret, opts)

	// Valid header, invalid payload base64
	validHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	token := validHeader + ".!!!.signature"

	_, err := validator(token)
	if err == nil {
		t.Error("expected error for invalid payload base64")
	}
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
	if err == nil {
		t.Error("expected error for invalid payload JSON")
	}
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
	if err == nil || !strings.Contains(err.Error(), "audience") {
		t.Errorf("expected 'missing audience' error, got: %v", err)
	}
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
	if err == nil || !strings.Contains(err.Error(), "invalid audience") {
		t.Errorf("expected 'invalid audience' error, got: %v", err)
	}
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
	if err == nil || !strings.Contains(err.Error(), "invalid audience format") {
		t.Errorf("expected 'invalid audience format' error, got: %v", err)
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token without exp should be valid
	validator := HS256Validator(secret, opts)
	validatedClaims, err := validator(token)
	if err != nil {
		t.Errorf("expected token without exp to be valid: %v", err)
	}
	if validatedClaims == nil {
		t.Error("expected claims to be returned")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token without nbf should be valid
	validator := HS256Validator(secret, opts)
	validatedClaims, err := validator(token)
	if err != nil {
		t.Errorf("expected token without nbf to be valid: %v", err)
	}
	if validatedClaims == nil {
		t.Error("expected claims to be returned")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate and check issuer wasn't added
	validatedClaims, err := store.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	hsClaims := validatedClaims.(HS256Claims)
	if _, ok := hsClaims["iss"]; ok {
		t.Error("expected no issuer claim when Issuer is empty")
	}
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
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate and check audience wasn't added
	validatedClaims, err := store.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	hsClaims := validatedClaims.(HS256Claims)
	if _, ok := hsClaims["aud"]; ok {
		t.Error("expected no audience claim when Audience is empty")
	}
}

func TestGetHS256Subject_Missing(t *testing.T) {
	claims := HS256Claims{
		// No "sub" claim
	}

	sub := GetHS256Subject(claims)
	if sub != "" {
		t.Errorf("expected empty subject, got %q", sub)
	}
}

func TestGetHS256Expiration_Missing(t *testing.T) {
	claims := HS256Claims{
		// No "exp" claim
	}

	exp := GetHS256Expiration(claims)
	if !exp.IsZero() {
		t.Errorf("expected zero time, got %v", exp)
	}
}

func TestHS256Store_Close(t *testing.T) {
	secret := []byte("my-secret-key-that-is-32-bytes-for-jwt!")
	opts := HS256Config{}
	store := NewHS256Store(secret, opts)

	err := store.Close()
	if err != nil {
		t.Errorf("unexpected error closing store: %v", err)
	}
}

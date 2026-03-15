package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// mockTokenStore is a test implementation of config.TokenStore
type mockTokenStore struct {
	validateFunc  func(ctx context.Context, token string) (config.JWTClaims, error)
	generateFunc  func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error)
	revokeFunc    func(ctx context.Context, claims map[string]any) error
	isRevoked     bool
	isRevokedFunc func(ctx context.Context, claims map[string]any) (bool, error)
}

func (m *mockTokenStore) Validate(ctx context.Context, token string) (config.JWTClaims, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return nil, errors.New("validator not configured")
}

func (m *mockTokenStore) Generate(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, claims, tokenType, ttl)
	}
	return "", errors.New("generator not configured")
}

func (m *mockTokenStore) Revoke(ctx context.Context, claims map[string]any) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, claims)
	}
	return nil
}

func (m *mockTokenStore) IsRevoked(ctx context.Context, claims map[string]any) (bool, error) {
	if m.isRevokedFunc != nil {
		return m.isRevokedFunc(ctx, claims)
	}
	return m.isRevoked, nil
}

func TestJWTAuth_MissingToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	reg := metrics.NewRegistry()
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(metrics.WithRegistry(req.Context(), reg))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Verify the metric was incremented with "invalid" label
	found := false
	for _, family := range reg.Gather() {
		if family.Name == "jwt_auth_requests_total" {
			for _, m := range family.Metrics {
				if m.Labels["result"] == "invalid" && m.Counter == 1 {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected jwt_auth_requests_total metric with result='invalid' to be 1")
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestJWTAuth_Success(t *testing.T) {
	expectedClaims := map[string]any{
		"sub":   "user123",
		"scope": "read write",
	}

	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return expectedClaims, nil
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwt := GetJWTClaims(r)
		if jwt.Raw() == nil {
			t.Error("expected claims in context")
		}

		token := GetJWTToken(r)
		if token != "valid-token" {
			t.Errorf("expected token 'valid-token', got %q", token)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "success" {
		t.Errorf("expected body 'success', got %q", rr.Body.String())
	}
}

func TestJWTAuth_ExemptPath(t *testing.T) {
	middleware := JWTAuth(config.JWTAuthConfig{
		ExemptPaths: []string{"/health"},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/health", http.StatusOK},
		{"/api/protected", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d for path %s, got %d", tt.wantStatus, tt.path, rr.Code)
			}
		})
	}
}

func TestJWTAuth_ExemptMethod(t *testing.T) {
	middleware := JWTAuth(config.JWTAuthConfig{
		ExemptMethods: []string{http.MethodHead},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		method     string
		wantStatus int
	}{
		{http.MethodHead, http.StatusOK},
		{http.MethodOptions, http.StatusOK},
		{http.MethodGet, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/protected", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d for method %s, got %d", tt.wantStatus, tt.method, rr.Code)
			}
		})
	}
}

func TestJWTAuth_RequiredClaims(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub": "user123",
			}, nil
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore:     store,
		RequiredClaims: []string{"sub", "iss"},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestJWTAuth_CustomErrorHandler(t *testing.T) {
	customHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"error":"custom error"}`))
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		ErrorHandler: customHandler,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}
}

func TestJWTAuth_OnSuccess(t *testing.T) {
	successCalled := false
	var receivedClaims config.JWTClaims

	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
		OnSuccess: func(r *http.Request, claims config.JWTClaims) {
			successCalled = true
			receivedClaims = claims
		},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !successCalled {
		t.Error("OnSuccess should have been called")
	}
	if receivedClaims == nil {
		t.Error("claims should have been passed to OnSuccess")
	}
}

func TestGetJWTClaims_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	jwt := GetJWTClaims(req)
	if jwt.Raw() != nil {
		t.Error("expected nil claims")
	}
}

func TestGetJWTToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetJWTToken(r)
		if token != "my-token" {
			t.Errorf("expected token 'my-token', got %q", token)
		}
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), JWTTokenContextKey, "my-token")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestGetJWTError(t *testing.T) {
	err := &JWTAuthError{
		Type:   "test-error",
		Title:  "Test Error",
		Status: http.StatusUnauthorized,
		Detail: "test detail",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := GetJWTError(r)
		if got != err {
			t.Error("expected error in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), JWTErrorContextKey, err)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestGetJWTClaimsSubject(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected string
	}{
		{
			name:     "has subject",
			claims:   map[string]any{"sub": "user123"},
			expected: "user123",
		},
		{
			name:     "missing subject",
			claims:   map[string]any{},
			expected: "",
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: "",
		},
		{
			name:     "non-map claims",
			claims:   "not-a-map",
			expected: "",
		},
		{
			name:     "HS256Claims type",
			claims:   HS256Claims{"sub": "user123"},
			expected: "user123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetJWTClaims(req).Subject()
			if got != tt.expected {
				t.Errorf("GetJWTClaimsSubject() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetJWTClaimsIssuer(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected string
	}{
		{
			name:     "has issuer",
			claims:   map[string]any{"iss": "my-issuer"},
			expected: "my-issuer",
		},
		{
			name:     "missing issuer",
			claims:   map[string]any{},
			expected: "",
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: "",
		},
		{
			name:     "non-map claims",
			claims:   "not-a-map",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetJWTClaims(req).Issuer()
			if got != tt.expected {
				t.Errorf("GetJWTClaimsIssuer() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetJWTClaimsAudience(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected []string
	}{
		{
			name:     "has audience string",
			claims:   map[string]any{"aud": "my-audience"},
			expected: []string{"my-audience"},
		},
		{
			name:     "has audience array",
			claims:   map[string]any{"aud": []string{"aud1", "aud2"}},
			expected: []string{"aud1", "aud2"},
		},
		{
			name:     "has audience any array",
			claims:   map[string]any{"aud": []any{"aud1", "aud2"}},
			expected: []string{"aud1", "aud2"},
		},
		{
			name:     "missing audience",
			claims:   map[string]any{},
			expected: nil,
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: nil,
		},
		{
			name:     "non-map claims",
			claims:   "not-a-map",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetJWTClaims(req).Audience()
			if !slicesEqual(got, tt.expected) {
				t.Errorf("GetJWTClaimsAudience() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetJWTClaimsJTI(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected string
	}{
		{
			name:     "has jti",
			claims:   map[string]any{"jti": "token-id-123"},
			expected: "token-id-123",
		},
		{
			name:     "missing jti",
			claims:   map[string]any{},
			expected: "",
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: "",
		},
		{
			name:     "non-map claims",
			claims:   "not-a-map",
			expected: "",
		},
		{
			name:     "HS256Claims type",
			claims:   HS256Claims{"jti": "token-id-456"},
			expected: "token-id-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetJWTClaims(req).JTI()
			if got != tt.expected {
				t.Errorf("GetJWTClaimsJTI() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetJWTClaimsScopes(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected []string
	}{
		{
			name:     "space-separated scopes",
			claims:   map[string]any{"scope": "read write delete"},
			expected: []string{"read", "write", "delete"},
		},
		{
			name:     "string slice scopes",
			claims:   map[string]any{"scope": []string{"read", "write"}},
			expected: []string{"read", "write"},
		},
		{
			name:     "any slice scopes",
			claims:   map[string]any{"scope": []any{"read", "write"}},
			expected: []string{"read", "write"},
		},
		{
			name:     "missing scope",
			claims:   map[string]any{},
			expected: nil,
		},
		{
			name:     "empty scope string",
			claims:   map[string]any{"scope": ""},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			scopes := GetJWTClaims(req).Scopes()
			if len(scopes) != len(tt.expected) {
				t.Errorf("GetJWTClaimsScopes() length = %d, want %d", len(scopes), len(tt.expected))
				return
			}
			for i := range scopes {
				if scopes[i] != tt.expected[i] {
					t.Errorf("GetJWTClaimsScopes()[%d] = %q, want %q", i, scopes[i], tt.expected[i])
				}
			}
		})
	}
}

func TestJWTClaims_HasScope(t *testing.T) {
	claims := map[string]any{"scope": "read write admin"}
	ctx := context.WithValue(context.Background(), JWTClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	jwt := GetJWTClaims(req)
	if !jwt.HasScope("read") {
		t.Error("should have 'read' scope")
	}
	if !jwt.HasScope("write") {
		t.Error("should have 'write' scope")
	}
	if !jwt.HasScope("admin") {
		t.Error("should have 'admin' scope")
	}
	if jwt.HasScope("delete") {
		t.Error("should not have 'delete' scope")
	}
}

func TestJWTClaims_HasScope_NoClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	jwt := GetJWTClaims(req)
	if jwt.HasScope("read") {
		t.Error("should return false with no claims")
	}
}

func TestJWTAuthError(t *testing.T) {
	err := &JWTAuthError{
		Type:   "test-error",
		Title:  "Test Error",
		Status: http.StatusBadRequest,
		Detail: "something went wrong",
	}

	if err.Error() != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", err.Error())
	}
}

func TestGenerateAccessToken_NoStore(t *testing.T) {
	cfg := config.JWTAuthConfig{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GenerateAccessToken(req, map[string]any{"sub": "user"}, cfg)
	if err == nil {
		t.Error("expected error when store not configured")
	}
}

func TestGenerateRefreshToken_NoStore(t *testing.T) {
	cfg := config.JWTAuthConfig{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GenerateRefreshToken(req, map[string]any{"sub": "user"}, cfg)
	if err == nil {
		t.Error("expected error when store not configured")
	}
}

func TestGenerateAccessToken_Success(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "generated-access-token", nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore:     store,
		AccessTokenTTL: 15 * time.Minute,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token, err := GenerateAccessToken(req, map[string]any{"sub": "user"}, cfg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if token != "generated-access-token" {
		t.Errorf("expected token 'generated-access-token', got %q", token)
	}
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "generated-refresh-token", nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore:      store,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token, err := GenerateRefreshToken(req, map[string]any{"sub": "user"}, cfg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if token != "generated-refresh-token" {
		t.Errorf("expected token 'generated-refresh-token', got %q", token)
	}
}

func TestGenerateAccessToken_StoreError(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "", errors.New("token generation failed")
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore:     store,
		AccessTokenTTL: 15 * time.Minute,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := GenerateAccessToken(req, map[string]any{"sub": "user"}, cfg)
	if err == nil {
		t.Error("expected error when generator fails")
	}
	if err.Error() != "token generation failed" {
		t.Errorf("expected error message 'token generation failed', got %q", err.Error())
	}
}

func TestGenerateRefreshToken_StoreError(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "", errors.New("token generation failed")
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore:      store,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := GenerateRefreshToken(req, map[string]any{"sub": "user"}, cfg)
	if err == nil {
		t.Error("expected error when generator fails")
	}
	if err.Error() != "token generation failed" {
		t.Errorf("expected error message 'token generation failed', got %q", err.Error())
	}
}

func TestRefreshTokenHandler(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			if token == "valid-refresh-token" {
				return map[string]any{
					"sub":  "user123",
					"type": config.TokenTypeRefresh,
				}, nil
			}
			return nil, errors.New("invalid token")
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			if tokenType == config.AccessToken {
				return "new-access-token", nil
			}
			return "new-refresh-token", nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore:      store,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["access_token"] != "new-access-token" {
		t.Errorf("expected access_token 'new-access-token', got %v", resp["access_token"])
	}
	if resp["refresh_token"] != "new-refresh-token" {
		t.Errorf("expected refresh_token 'new-refresh-token', got %v", resp["refresh_token"])
	}
	if resp["token_type"] != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got %v", resp["token_type"])
	}
}

func TestRefreshTokenHandler_InvalidMethod(t *testing.T) {
	cfg := config.JWTAuthConfig{}
	handler := RefreshTokenHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/refresh", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}

	// Test JSON response
	req.Header.Set("Accept", "application/json")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).IsProblemDetail().ProblemDetailDetail("Method not allowed")

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/auth/refresh", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).Header("Content-Type", "text/plain; charset=utf-8")
}

func TestRefreshTokenHandler_MissingToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}
	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
}

func TestRefreshTokenHandler_InvalidToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"invalid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestRefreshTokenHandler_NotRefreshToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": "access",
			}, nil
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"access-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
}

func TestRefreshTokenHandler_TokenRevoked(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
				"jti":  "token-id-123",
			}, nil
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
		isRevoked: true,
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := RefreshTokenHandler(cfg)

	// Test JSON response
	body := `{"refresh_token":"revoked-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestRefreshTokenHandler_TokenAllowed(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
				"jti":  "valid-token-id",
			}, nil
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			if tokenType == config.AccessToken {
				return "new-access-token", nil
			}
			return "new-refresh-token", nil
		},
		isRevoked: false,
	}

	cfg := config.JWTAuthConfig{
		TokenStore:     store,
		AccessTokenTTL: 15 * time.Minute,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestLogoutTokenHandler(t *testing.T) {
	revokeCalled := false
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
				"jti":  "token-id-123",
			}, nil
		},
		revokeFunc: func(ctx context.Context, claims map[string]any) error {
			revokeCalled = true
			return nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !revokeCalled {
		t.Error("Revoke should have been called")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["message"] != "logged out successfully" {
		t.Errorf("expected message 'logged out successfully', got %v", resp["message"])
	}
}

func TestLogoutTokenHandler_InvalidMethod(t *testing.T) {
	cfg := config.JWTAuthConfig{}
	handler := LogoutTokenHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}

	// Test JSON response
	req.Header.Set("Accept", "application/json")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).IsProblemDetail().ProblemDetailDetail("Method not allowed")

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).Header("Content-Type", "text/plain; charset=utf-8")
}

func TestLogoutTokenHandler_NoTokenStore(t *testing.T) {
	cfg := config.JWTAuthConfig{}
	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"some-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestLogoutTokenHandler_MissingToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}
	handler := LogoutTokenHandler(cfg)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
}

func TestLogoutTokenHandler_InvalidToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"invalid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestLogoutTokenHandler_NotRefreshToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": "access",
			}, nil
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"access-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
}

func TestLogoutTokenHandler_RevokeError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
				"jti":  "token-id-123",
			}, nil
		},
		revokeFunc: func(ctx context.Context, claims map[string]any) error {
			return errors.New("database error")
		},
	}

	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}

	handler := LogoutTokenHandler(cfg)

	// Test JSON response
	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestGetJWTClaimsExpiration(t *testing.T) {
	now := time.Now()
	expUnix := now.Add(15 * time.Minute).Unix()

	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected time.Time
	}{
		{
			name:     "float64 exp",
			claims:   map[string]any{"exp": float64(expUnix)},
			expected: time.Unix(expUnix, 0),
		},
		{
			name:     "int64 exp",
			claims:   map[string]any{"exp": expUnix},
			expected: time.Unix(expUnix, 0),
		},
		{
			name:     "missing exp",
			claims:   map[string]any{},
			expected: time.Time{},
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), JWTClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetJWTClaims(req).Expiration()
			if !got.Equal(tt.expected) {
				t.Errorf("GetJWTClaimsExpiration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAddExpirationToClaims(t *testing.T) {
	ttl := 15 * time.Minute
	before := time.Now()

	claims := map[string]any{"sub": "user123"}
	result := addExpirationToClaims(claims, ttl)

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result should be map[string]any")
	}

	if resultMap["sub"] != "user123" {
		t.Errorf("expected sub = 'user123', got %v", resultMap["sub"])
	}

	exp, ok := resultMap["exp"].(int64)
	if !ok {
		t.Fatal("exp should be int64")
	}

	after := time.Now().Add(ttl)
	if exp < before.Add(ttl).Unix() || exp > after.Unix() {
		t.Errorf("exp %d not in expected range [%d, %d]", exp, before.Add(ttl).Unix(), after.Unix())
	}

	if _, exists := claims["exp"]; exists {
		t.Error("original claims should not be modified")
	}
}

func TestAddTypeToClaims(t *testing.T) {
	claims := map[string]any{"sub": "user123"}
	result := addTypeToClaims(claims, "refresh")

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("result should be map[string]any")
	}

	if resultMap["type"] != "refresh" {
		t.Errorf("expected type = 'refresh', got %v", resultMap["type"])
	}

	if _, exists := claims["type"]; exists {
		t.Error("original claims should not be modified")
	}
}

func TestHasClaim(t *testing.T) {
	claims := map[string]any{"sub": "user123", "iss": "test"}

	if !hasClaim(claims, "sub") {
		t.Error("should have 'sub' claim")
	}
	if !hasClaim(claims, "iss") {
		t.Error("should have 'iss' claim")
	}
	if hasClaim(claims, "aud") {
		t.Error("should not have 'aud' claim")
	}

	// Test with HS256Claims type
	hsClaims := HS256Claims{"sub": "user123", "iss": "test"}
	if !hasClaim(hsClaims, "sub") {
		t.Error("should have 'sub' claim in HS256Claims")
	}
	if !hasClaim(hsClaims, "iss") {
		t.Error("should have 'iss' claim in HS256Claims")
	}
	if hasClaim(hsClaims, "aud") {
		t.Error("should not have 'aud' claim in HS256Claims")
	}
}

// Define a custom map type to simulate jwt.MapClaims from golang-jwt/jwt
type customMapClaims map[string]any

func TestHasClaim_CustomMapType(t *testing.T) {
	// Test with custom map type (like jwt.MapClaims)
	claims := customMapClaims{"sub": "user123", "iss": "test"}

	if !hasClaim(claims, "sub") {
		t.Error("should have 'sub' claim in custom map type")
	}
	if !hasClaim(claims, "iss") {
		t.Error("should have 'iss' claim in custom map type")
	}
	if hasClaim(claims, "aud") {
		t.Error("should not have 'aud' claim in custom map type")
	}
}

func TestGetStringClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		key      string
		expected string
	}{
		{
			name:     "string value",
			claims:   map[string]any{"sub": "user123"},
			key:      "sub",
			expected: "user123",
		},
		{
			name:     "string slice first value",
			claims:   map[string]any{"aud": []string{"aud1", "aud2"}},
			key:      "aud",
			expected: "aud1",
		},
		{
			name:     "any slice first value",
			claims:   map[string]any{"aud": []any{"aud1", "aud2"}},
			key:      "aud",
			expected: "aud1",
		},
		{
			name:     "missing key",
			claims:   map[string]any{},
			key:      "sub",
			expected: "",
		},
		{
			name:     "non-map claims",
			claims:   "not-a-map",
			key:      "sub",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStringClaim(tt.claims, tt.key)
			if got != tt.expected {
				t.Errorf("getStringClaim() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetMapClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		key      string
		expected any
	}{
		{
			name:     "map[string]any",
			claims:   map[string]any{"sub": "user123"},
			key:      "sub",
			expected: "user123",
		},
		{
			name:     "custom map type",
			claims:   customMapClaims{"sub": "user456"},
			key:      "sub",
			expected: "user456",
		},
		{
			name:     "missing key",
			claims:   map[string]any{},
			key:      "sub",
			expected: nil,
		},
		{
			name:     "nil claims",
			claims:   nil,
			key:      "sub",
			expected: nil,
		},
		{
			name:     "non-map claims",
			claims:   "not-a-map",
			key:      "sub",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMapClaim(tt.claims, tt.key)
			if got != tt.expected {
				t.Errorf("getMapClaim() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractStringValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{name: "string", value: "hello", expected: "hello"},
		{name: "nil", value: nil, expected: ""},
		{name: "string slice", value: []string{"a", "b"}, expected: "a"},
		{name: "any slice", value: []any{"x", "y"}, expected: "x"},
		{name: "empty slice", value: []string{}, expected: ""},
		{name: "int", value: 42, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStringValue(tt.value)
			if got != tt.expected {
				t.Errorf("extractStringValue() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetJWTClaims(t *testing.T) {
	claims := map[string]any{"sub": "user123"}
	ctx := context.WithValue(context.Background(), JWTClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	jwt := GetJWTClaims(req)
	if jwt.Raw() == nil {
		t.Error("GetJWTClaims should return claims from context")
	}

	// Compare by checking we can access the same data
	wrappedClaims, ok := jwt.Raw().(map[string]any)
	if !ok {
		t.Fatal("Raw() should return the original claims type")
	}
	if wrappedClaims["sub"] != "user123" {
		t.Error("GetJWTClaims should preserve the claims data")
	}

	// Test direct accessor
	if jwt.Subject() != "user123" {
		t.Errorf("Subject() = %q, want 'user123'", jwt.Subject())
	}
}

func TestClaimsWrapper_Subject(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected string
	}{
		{
			name:     "has subject",
			claims:   map[string]any{"sub": "user123"},
			expected: "user123",
		},
		{
			name:     "missing subject",
			claims:   map[string]any{},
			expected: "",
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: "",
		},
		{
			name:     "HS256Claims",
			claims:   HS256Claims{"sub": "user456"},
			expected: "user456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwt := JWTClaims{claims: tt.claims}
			got := jwt.Subject()
			if got != tt.expected {
				t.Errorf("Subject() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestClaimsWrapper_Issuer(t *testing.T) {
	claims := map[string]any{"iss": "my-issuer"}
	jwt := JWTClaims{claims: claims}

	if got := jwt.Issuer(); got != "my-issuer" {
		t.Errorf("Issuer() = %q, want 'my-issuer'", got)
	}
}

func TestClaimsWrapper_Audience(t *testing.T) {
	claims := map[string]any{"aud": "my-audience"}
	jwt := JWTClaims{claims: claims}

	if got := jwt.Audience(); !slicesEqual(got, []string{"my-audience"}) {
		t.Errorf("Audience() = %v, want ['my-audience']", got)
	}
}

func TestClaimsWrapper_Audience_Array(t *testing.T) {
	claims := map[string]any{"aud": []string{"aud1", "aud2"}}
	jwt := JWTClaims{claims: claims}

	if got := jwt.Audience(); !slicesEqual(got, []string{"aud1", "aud2"}) {
		t.Errorf("Audience() = %v, want ['aud1', 'aud2']", got)
	}
}

func TestClaimsWrapper_HasAudience(t *testing.T) {
	claims := map[string]any{"aud": []string{"aud1", "aud2"}}
	jwt := JWTClaims{claims: claims}

	if !jwt.HasAudience("aud1") {
		t.Error("HasAudience('aud1') = false, want true")
	}
	if !jwt.HasAudience("aud2") {
		t.Error("HasAudience('aud2') = false, want true")
	}
	if jwt.HasAudience("aud3") {
		t.Error("HasAudience('aud3') = true, want false")
	}
}

func TestClaimsWrapper_JTI(t *testing.T) {
	claims := map[string]any{"jti": "token-id-123"}
	jwt := JWTClaims{claims: claims}

	if got := jwt.JTI(); got != "token-id-123" {
		t.Errorf("JTI() = %q, want 'token-id-123'", got)
	}
}

func TestClaimsWrapper_Expiration(t *testing.T) {
	now := time.Now()
	expUnix := now.Add(time.Hour).Unix()

	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected time.Time
	}{
		{
			name:     "float64 exp",
			claims:   map[string]any{"exp": float64(expUnix)},
			expected: time.Unix(expUnix, 0),
		},
		{
			name:     "int64 exp",
			claims:   map[string]any{"exp": expUnix},
			expected: time.Unix(expUnix, 0),
		},
		{
			name:     "missing exp",
			claims:   map[string]any{},
			expected: time.Time{},
		},
		{
			name:     "nil claims",
			claims:   nil,
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwt := JWTClaims{claims: tt.claims}
			got := jwt.Expiration()
			if !got.Equal(tt.expected) {
				t.Errorf("Expiration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaimsWrapper_Scopes(t *testing.T) {
	tests := []struct {
		name     string
		claims   config.JWTClaims
		expected []string
	}{
		{
			name:     "space-separated",
			claims:   map[string]any{"scope": "read write delete"},
			expected: []string{"read", "write", "delete"},
		},
		{
			name:     "string slice",
			claims:   map[string]any{"scope": []string{"read", "write"}},
			expected: []string{"read", "write"},
		},
		{
			name:     "any slice",
			claims:   map[string]any{"scope": []any{"read", "write"}},
			expected: []string{"read", "write"},
		},
		{
			name:     "missing scope",
			claims:   map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwt := JWTClaims{claims: tt.claims}
			got := jwt.Scopes()
			if len(got) != len(tt.expected) {
				t.Errorf("Scopes() length = %d, want %d", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("Scopes()[%d] = %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestClaimsWrapper_HasScope(t *testing.T) {
	claims := map[string]any{"scope": "read write admin"}
	jwt := JWTClaims{claims: claims}

	if !jwt.HasScope("read") {
		t.Error("should have 'read' scope")
	}
	if !jwt.HasScope("write") {
		t.Error("should have 'write' scope")
	}
	if !jwt.HasScope("admin") {
		t.Error("should have 'admin' scope")
	}
	if jwt.HasScope("delete") {
		t.Error("should not have 'delete' scope")
	}
}

func TestClaimsWrapper_HasScope_NoClaims(t *testing.T) {
	jwt := JWTClaims{claims: nil}
	if jwt.HasScope("read") {
		t.Error("should return false with nil claims")
	}
}

func TestGetJWTToken_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token := GetJWTToken(req)
	if token != "" {
		t.Errorf("expected empty string, got %q", token)
	}
}

func TestGetJWTError_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	err := GetJWTError(req)
	if err != nil {
		t.Error("expected nil error")
	}
}

func TestGetJWTError_NotJWTError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), JWTErrorContextKey, errors.New("some error"))
	err := GetJWTError(req.WithContext(ctx))
	if err != nil {
		t.Error("expected nil for non-JWTAuthError")
	}
}

func TestExpiration_NonMapClaims(t *testing.T) {
	// Test with non-map claims type
	jwt := JWTClaims{claims: "not a map"}
	exp := jwt.Expiration()
	if !exp.IsZero() {
		t.Error("expected zero time for non-map claims")
	}
}

func TestExpiration_Int64(t *testing.T) {
	claims := map[string]any{"exp": int64(time.Now().Add(time.Hour).Unix())}
	jwt := JWTClaims{claims: claims}
	exp := jwt.Expiration()
	if exp.IsZero() {
		t.Error("expected valid expiration time")
	}
}

func TestScopes_NonMapClaims(t *testing.T) {
	jwt := JWTClaims{claims: "not a map"}
	scopes := jwt.Scopes()
	if scopes != nil {
		t.Error("expected nil for non-map claims")
	}
}

func TestScopes_EmptyString(t *testing.T) {
	jwt := JWTClaims{claims: map[string]any{"scope": ""}}
	scopes := jwt.Scopes()
	if len(scopes) != 0 {
		t.Errorf("expected empty slice, got %v", scopes)
	}
}

func TestDefaultJWTErrorHandler_NoError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Call handler without setting error in context
	defaultJWTErrorHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestExtractBearerToken_NoBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")

	token := extractBearerToken(req)
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractBearerToken_EmptyAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token := extractBearerToken(req)
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestDeepCopyMap_Nil(t *testing.T) {
	result := deepCopyMap(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestDeepCopyMap_Nested(t *testing.T) {
	original := map[string]any{
		"user": map[string]any{
			"name": "alice",
			"tags": []any{"admin", "user"},
		},
	}

	copied := deepCopyMap(original)

	// Modify original
	original["user"].(map[string]any)["name"] = "bob"
	original["user"].(map[string]any)["tags"].([]any)[0] = "superadmin"

	// Check copy wasn't affected
	user := copied["user"].(map[string]any)
	if user["name"] != "alice" {
		t.Error("deep copy failed - name was modified")
	}
	tags := user["tags"].([]any)
	if tags[0] != "admin" {
		t.Error("deep copy failed - tags were modified")
	}
}

func TestDeepCopySlice_Nil(t *testing.T) {
	result := deepCopySlice(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestAddExpirationToClaims_NonMap(t *testing.T) {
	claims := "not a map"
	result := addExpirationToClaims(claims, time.Hour)
	if result != claims {
		t.Error("expected original claims for non-map type")
	}
}

func TestAddTypeToClaims_NonMap(t *testing.T) {
	claims := "not a map"
	result := addTypeToClaims(claims, "refresh")
	if result != claims {
		t.Error("expected original claims for non-map type")
	}
}

func TestGenerateAccessToken_DefaultTTL(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}
	cfg := config.JWTAuthConfig{
		TokenStore:     store,
		AccessTokenTTL: 0, // Should use default
	}
	claims := map[string]any{"sub": "user123"}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	token, err := GenerateAccessToken(req, claims, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if token != "token" {
		t.Errorf("expected 'token', got %q", token)
	}
}

func TestRefreshTokenHandler_StoreError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
			}, nil
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			if tokenType == config.AccessToken {
				return "", errors.New("generate failed")
			}
			return "refresh", nil
		},
	}
	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestLogoutTokenHandler_AlreadyRevoked(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
			}, nil
		},
		isRevoked: true,
	}
	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}
	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

func TestJWTAuth_RevokedToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
		isRevoked: true,
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer revoked-token")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer revoked-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestJWTAuth_RefreshTokenAsAccessToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
			}, nil
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer refresh-token")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer refresh-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

func TestJWTAuth_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			if token == "valid-token" {
				return map[string]any{"sub": "user123"}, nil
			}
			return nil, errors.New("invalid token")
		},
	}
	mw := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       true,
		PathLabelFunc: func(p string) string { return p },
	})
	wrapped := metricsMw(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// Request without token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr1.Code)
	}

	// Request with valid token
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer valid-token")
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr2.Code)
	}

	// Check metrics
	families := reg.Gather()
	var counter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "jwt_auth_requests_total" {
			counter = &f
			break
		}
	}
	if counter == nil {
		t.Fatal("expected jwt_auth_requests_total metric")
	}

	// Should have metrics for both valid and invalid
	results := make(map[string]int)
	for _, m := range counter.Metrics {
		results[m.Labels["result"]]++
	}
	if results["missing"] != 1 {
		t.Errorf("expected 1 missing, got %d", results["missing"])
	}
	if results["valid"] != 1 {
		t.Errorf("expected 1 valid, got %d", results["valid"])
	}
}

// Test for IsRevoked error path in JWTAuth middleware
func TestJWTAuth_IsRevokedCheckError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
		isRevokedFunc: func(ctx context.Context, claims map[string]any) (bool, error) {
			return false, errors.New("database connection failed")
		},
	}

	middleware := JWTAuth(config.JWTAuthConfig{
		TokenStore: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var errResp JWTAuthError
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if !strings.Contains(errResp.Title, "Token Revocation Check Failed") {
		t.Errorf("expected 'Token Revocation Check Failed', got %q", errResp.Title)
	}

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !strings.Contains(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain, got %s", rr.Header().Get("Content-Type"))
	}
}

// Test for IsRevoked error path in RefreshTokenHandler
func TestRefreshTokenHandler_IsRevokedCheckError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
			}, nil
		},
		isRevokedFunc: func(ctx context.Context, claims map[string]any) (bool, error) {
			return false, errors.New("database connection failed")
		},
	}
	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// Test for getStringClaim with []any slice containing non-string elements
func TestGetStringClaim_AnySliceWithNonString(t *testing.T) {
	claims := map[string]any{
		"aud": []any{123, "aud2"}, // First element is not a string
	}

	result := getStringClaim(claims, "aud")
	if result != "" {
		t.Errorf("expected empty string for non-string first element, got %q", result)
	}

	// Test with empty []any slice
	claims2 := map[string]any{
		"aud": []any{},
	}

	result2 := getStringClaim(claims2, "aud")
	if result2 != "" {
		t.Errorf("expected empty string for empty slice, got %q", result2)
	}
}

// Test for getStringClaim with reflection path for non-map claims
func TestGetStringClaim_ReflectionPath(t *testing.T) {
	// Use a custom map type that requires reflection
	claims := customMapClaims{
		"sub": "user123",
	}

	result := getStringClaim(claims, "sub")
	if result != "user123" {
		t.Errorf("expected 'user123', got %q", result)
	}

	// Test missing key via reflection
	result2 := getStringClaim(claims, "missing")
	if result2 != "" {
		t.Errorf("expected empty string for missing key, got %q", result2)
	}
}

// Test for deepCopySlice with nested []any
func TestDeepCopySlice_NestedSlice(t *testing.T) {
	original := []any{
		"simple",
		[]any{"nested1", "nested2"},
		map[string]any{"key": "value"},
	}

	copied := deepCopySlice(original)

	// Modify original
	original[0] = "modified"
	original[1].([]any)[0] = "modified-nested"
	original[2].(map[string]any)["key"] = "modified-value"

	// Check copy wasn't affected
	if copied[0] != "simple" {
		t.Error("deep copy failed - simple element was modified")
	}
	nested := copied[1].([]any)
	if nested[0] != "nested1" {
		t.Error("deep copy failed - nested slice element was modified")
	}
	m := copied[2].(map[string]any)
	if m["key"] != "value" {
		t.Error("deep copy failed - nested map element was modified")
	}
}

// Test for addExpirationToClaims with HS256Claims
func TestAddExpirationToClaims_HS256Claims(t *testing.T) {
	ttl := 15 * time.Minute
	claims := HS256Claims{"sub": "user123"}

	result := addExpirationToClaims(claims, ttl)

	resultMap, ok := result.(HS256Claims)
	if !ok {
		t.Fatal("result should be HS256Claims")
	}

	if resultMap["sub"] != "user123" {
		t.Errorf("expected sub = 'user123', got %v", resultMap["sub"])
	}

	if _, exists := resultMap["exp"]; !exists {
		t.Error("exp claim should be added")
	}

	// Original should not be modified
	if _, exists := claims["exp"]; exists {
		t.Error("original claims should not be modified")
	}
}

// Test for addTypeToClaims with HS256Claims
func TestAddTypeToClaims_HS256Claims(t *testing.T) {
	claims := HS256Claims{"sub": "user123"}

	result := addTypeToClaims(claims, "refresh")

	resultMap, ok := result.(HS256Claims)
	if !ok {
		t.Fatal("result should be HS256Claims")
	}

	if resultMap["type"] != "refresh" {
		t.Errorf("expected type = 'refresh', got %v", resultMap["type"])
	}

	// Original should not be modified
	if _, exists := claims["type"]; exists {
		t.Error("original claims should not be modified")
	}
}

// Test for RefreshTokenHandler GenerateRefreshToken error
func TestRefreshTokenHandler_GenerateRefreshTokenError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (config.JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": config.TokenTypeRefresh,
			}, nil
		},
		generateFunc: func(ctx context.Context, claims config.JWTClaims, tokenType config.TokenType, ttl time.Duration) (string, error) {
			if tokenType == config.RefreshToken {
				return "", errors.New("refresh token generation failed")
			}
			return "access-token", nil
		},
		isRevokedFunc: func(ctx context.Context, claims map[string]any) (bool, error) {
			return false, nil
		},
		revokeFunc: func(ctx context.Context, claims map[string]any) error {
			return nil
		},
	}
	cfg := config.JWTAuthConfig{
		TokenStore: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

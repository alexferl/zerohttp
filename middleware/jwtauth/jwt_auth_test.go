package jwtauth

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
	"github.com/alexferl/zerohttp/httpx"
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

// mockTokenStore is a test implementation of TokenStore
type mockTokenStore struct {
	validateFunc  func(ctx context.Context, token string) (JWTClaims, error)
	generateFunc  func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error)
	revokeFunc    func(ctx context.Context, claims map[string]any) error
	isRevoked     bool
	isRevokedFunc func(ctx context.Context, claims map[string]any) (bool, error)
}

func (m *mockTokenStore) Validate(ctx context.Context, token string) (JWTClaims, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, token)
	}
	return nil, errors.New("validator not configured")
}

func (m *mockTokenStore) Generate(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
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

func (m *mockTokenStore) Close() error {
	return nil
}

func TestJWTAuth_MissingToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
	}

	middleware := New(Config{
		Store: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	middleware := New(Config{
		Store: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	reg := metrics.NewRegistry()
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer invalid-token")
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	req = req.WithContext(metrics.WithRegistry(req.Context(), reg))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

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
	zhtest.AssertTrue(t, found)

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer invalid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

func TestJWTAuth_Success(t *testing.T) {
	expectedClaims := map[string]any{
		"sub":   "user123",
		"scope": "read write",
	}

	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return expectedClaims, nil
		},
	}

	middleware := New(Config{
		Store: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwt := GetClaims(r)
		zhtest.AssertNotNil(t, jwt.Raw())

		token := GetToken(r)
		zhtest.AssertEqual(t, "valid-token", token)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
	zhtest.AssertEqual(t, "success", strings.TrimSpace(rr.Body.String()))
}

func TestJWTAuth_ExcludedPath(t *testing.T) {
	middleware := New(Config{
		ExcludedPaths: []string{"/health"},
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
			zhtest.AssertEqual(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestJWTAuth_ExcludedMethod(t *testing.T) {
	middleware := New(Config{
		ExcludedMethods: []string{http.MethodHead},
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
			zhtest.AssertEqual(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestJWTAuth_RequiredClaims(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub": "user123",
			}, nil
		},
	}

	middleware := New(Config{
		Store:          store,
		RequiredClaims: []string{"sub", "iss"},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusForbidden, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

func TestJWTAuth_CustomErrorHandler(t *testing.T) {
	customHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"error":"custom error"}`))
	}

	middleware := New(Config{
		ErrorHandler: customHandler,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusTeapot, rr.Code)
}

func TestJWTAuth_OnSuccess(t *testing.T) {
	successCalled := false
	var receivedClaims JWTClaims

	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
	}

	middleware := New(Config{
		Store: store,
		OnSuccess: func(r *http.Request, claims JWTClaims) {
			successCalled = true
			receivedClaims = claims
		},
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, successCalled)
	zhtest.AssertNotNil(t, receivedClaims)
}

func TestGetJWTClaims_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	jwt := GetClaims(req)
	zhtest.AssertNil(t, jwt.Raw())
}

func TestGetJWTToken(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetToken(r)
		zhtest.AssertEqual(t, "my-token", token)
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), TokenContextKey, "my-token")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestGetJWTError(t *testing.T) {
	err := &AuthError{
		Type:   "test-error",
		Title:  "Test Error",
		Status: http.StatusUnauthorized,
		Detail: "test detail",
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := GetError(r)
		zhtest.AssertEqual(t, err, got)
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.WithValue(context.Background(), ErrorContextKey, err)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestGetJWTClaimsSubject(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			ctx := context.WithValue(context.Background(), ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetClaims(req).Subject()
			zhtest.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestGetJWTClaimsIssuer(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			ctx := context.WithValue(context.Background(), ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetClaims(req).Issuer()
			zhtest.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestGetJWTClaimsAudience(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			ctx := context.WithValue(context.Background(), ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetClaims(req).Audience()
			zhtest.AssertTrue(t, slicesEqual(got, tt.expected))
		})
	}
}

func TestGetJWTClaimsJTI(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			ctx := context.WithValue(context.Background(), ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetClaims(req).JTI()
			zhtest.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestGetJWTClaimsScopes(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			ctx := context.WithValue(context.Background(), ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			scopes := GetClaims(req).Scopes()
			zhtest.AssertEqual(t, len(tt.expected), len(scopes))
			for i := range scopes {
				zhtest.AssertEqual(t, tt.expected[i], scopes[i])
			}
		})
	}
}

func TestJWTClaims_HasScope(t *testing.T) {
	claims := map[string]any{"scope": "read write admin"}
	ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	jwt := GetClaims(req)
	zhtest.AssertTrue(t, jwt.HasScope("read"))
	zhtest.AssertTrue(t, jwt.HasScope("write"))
	zhtest.AssertTrue(t, jwt.HasScope("admin"))
	zhtest.AssertFalse(t, jwt.HasScope("delete"))
}

func TestJWTClaims_HasScope_NoClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	jwt := GetClaims(req)
	zhtest.AssertFalse(t, jwt.HasScope("read"))
}

func TestJWTAuthError(t *testing.T) {
	err := &AuthError{
		Type:   "test-error",
		Title:  "Test Error",
		Status: http.StatusBadRequest,
		Detail: "something went wrong",
	}

	zhtest.AssertEqual(t, "something went wrong", err.Error())
}

func TestGenerateAccessToken_NoStore(t *testing.T) {
	cfg := Config{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GenerateAccessToken(req, map[string]any{"sub": "user"}, cfg)
	zhtest.AssertError(t, err)
}

func TestGenerateRefreshToken_NoStore(t *testing.T) {
	cfg := Config{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GenerateRefreshToken(req, map[string]any{"sub": "user"}, cfg)
	zhtest.AssertError(t, err)
}

func TestGenerateAccessToken_Success(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "generated-access-token", nil
		},
	}

	cfg := Config{
		Store:          store,
		AccessTokenTTL: 15 * time.Minute,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token, err := GenerateAccessToken(req, map[string]any{"sub": "user"}, cfg)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "generated-access-token", token)
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "generated-refresh-token", nil
		},
	}

	cfg := Config{
		Store:           store,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token, err := GenerateRefreshToken(req, map[string]any{"sub": "user"}, cfg)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "generated-refresh-token", token)
}

func TestGenerateAccessToken_StoreError(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "", errors.New("token generation failed")
		},
	}

	cfg := Config{
		Store:          store,
		AccessTokenTTL: 15 * time.Minute,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := GenerateAccessToken(req, map[string]any{"sub": "user"}, cfg)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "token generation failed")
}

func TestGenerateRefreshToken_StoreError(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "", errors.New("token generation failed")
		},
	}

	cfg := Config{
		Store:           store,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := GenerateRefreshToken(req, map[string]any{"sub": "user"}, cfg)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "token generation failed")
}

func TestRefreshTokenHandler(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			if token == "valid-refresh-token" {
				return map[string]any{
					"sub":  "user123",
					"type": TokenTypeRefresh,
				}, nil
			}
			return nil, errors.New("invalid token")
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			if tokenType == AccessToken {
				return "new-access-token", nil
			}
			return "new-refresh-token", nil
		},
	}

	cfg := Config{
		Store:           store,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)

	var resp map[string]any
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	zhtest.AssertEqual(t, "new-access-token", resp["access_token"])
	zhtest.AssertEqual(t, "new-refresh-token", resp["refresh_token"])
	zhtest.AssertEqual(t, "Bearer", resp["token_type"])
}

func TestRefreshTokenHandler_InvalidMethod(t *testing.T) {
	cfg := Config{}
	handler := RefreshTokenHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/refresh", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusMethodNotAllowed, rr.Code)

	// Test JSON response
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).IsProblemDetail().ProblemDetailDetail("Method not allowed")

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/auth/refresh", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
}

func TestRefreshTokenHandler_MissingToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}
	cfg := Config{
		Store: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestRefreshTokenHandler_InvalidToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}

	cfg := Config{
		Store: store,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"invalid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestRefreshTokenHandler_NotRefreshToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": "access",
			}, nil
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}

	cfg := Config{
		Store: store,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"access-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestRefreshTokenHandler_TokenRevoked(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
				"jti":  "token-id-123",
			}, nil
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
		isRevoked: true,
	}

	cfg := Config{
		Store: store,
	}

	handler := RefreshTokenHandler(cfg)

	// Test JSON response
	body := `{"refresh_token":"revoked-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

	// Test plain text response
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

func TestRefreshTokenHandler_TokenAllowed(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
				"jti":  "valid-token-id",
			}, nil
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			if tokenType == AccessToken {
				return "new-access-token", nil
			}
			return "new-refresh-token", nil
		},
		isRevoked: false,
	}

	cfg := Config{
		Store:          store,
		AccessTokenTTL: 15 * time.Minute,
	}

	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusOK, rr.Code)
}

func TestLogoutTokenHandler(t *testing.T) {
	revokeCalled := false
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
				"jti":  "token-id-123",
			}, nil
		},
		revokeFunc: func(ctx context.Context, claims map[string]any) error {
			revokeCalled = true
			return nil
		},
	}

	cfg := Config{
		Store: store,
	}

	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertTrue(t, revokeCalled)
	zhtest.AssertEqual(t, http.StatusOK, rr.Code)

	var resp map[string]any
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	zhtest.AssertEqual(t, "logged out successfully", resp["message"])
}

func TestLogoutTokenHandler_InvalidMethod(t *testing.T) {
	cfg := Config{}
	handler := LogoutTokenHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusMethodNotAllowed, rr.Code)

	// Test JSON response
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).IsProblemDetail().ProblemDetailDetail("Method not allowed")

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertWith(t, rr).Header(httpx.HeaderContentType, "text/plain; charset=utf-8")
}

func TestLogoutTokenHandler_NoTokenStore(t *testing.T) {
	cfg := Config{}
	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"some-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestLogoutTokenHandler_MissingToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	cfg := Config{
		Store: store,
	}
	handler := LogoutTokenHandler(cfg)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestLogoutTokenHandler_InvalidToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return nil, errors.New("invalid token")
		},
	}

	cfg := Config{
		Store: store,
	}

	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"invalid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestLogoutTokenHandler_NotRefreshToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": "access",
			}, nil
		},
	}

	cfg := Config{
		Store: store,
	}

	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"access-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestLogoutTokenHandler_RevokeError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
				"jti":  "token-id-123",
			}, nil
		},
		revokeFunc: func(ctx context.Context, claims map[string]any) error {
			return errors.New("database error")
		},
	}

	cfg := Config{
		Store: store,
	}

	handler := LogoutTokenHandler(cfg)

	// Test JSON response
	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusInternalServerError, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

	// Test plain text response
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), httpx.MIMETextPlain)
}

func TestGetJWTClaimsExpiration(t *testing.T) {
	now := time.Now()
	expUnix := now.Add(15 * time.Minute).Unix()

	tests := []struct {
		name     string
		claims   JWTClaims
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
			ctx := context.WithValue(context.Background(), ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			got := GetClaims(req).Expiration()
			zhtest.AssertTrue(t, got.Equal(tt.expected))
		})
	}
}

func TestAddExpirationToClaims(t *testing.T) {
	ttl := 15 * time.Minute
	before := time.Now()

	claims := map[string]any{"sub": "user123"}
	result := addExpirationToClaims(claims, ttl)

	resultMap, ok := result.(map[string]any)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, "user123", resultMap["sub"])

	exp, ok := resultMap["exp"].(int64)
	zhtest.AssertTrue(t, ok)

	after := time.Now().Add(ttl)
	zhtest.AssertTrue(t, exp >= before.Add(ttl).Unix() && exp <= after.Unix())

	_, exists := claims["exp"]
	zhtest.AssertFalse(t, exists)
}

func TestAddTypeToClaims(t *testing.T) {
	claims := map[string]any{"sub": "user123"}
	result := addTypeToClaims(claims, "refresh")

	resultMap, ok := result.(map[string]any)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, "refresh", resultMap["type"])

	_, exists := claims["type"]
	zhtest.AssertFalse(t, exists)
}

func TestHasClaim(t *testing.T) {
	claims := map[string]any{"sub": "user123", "iss": "test"}

	zhtest.AssertTrue(t, hasClaim(claims, "sub"))
	zhtest.AssertTrue(t, hasClaim(claims, "iss"))
	zhtest.AssertFalse(t, hasClaim(claims, "aud"))

	// Test with HS256Claims type
	hsClaims := HS256Claims{"sub": "user123", "iss": "test"}
	zhtest.AssertTrue(t, hasClaim(hsClaims, "sub"))
	zhtest.AssertTrue(t, hasClaim(hsClaims, "iss"))
	zhtest.AssertFalse(t, hasClaim(hsClaims, "aud"))
}

// Define a custom map type to simulate jwt.MapClaims from golang-jwt/jwt
type customMapClaims map[string]any

func TestHasClaim_CustomMapType(t *testing.T) {
	// Test with custom map type (like jwt.MapClaims)
	claims := customMapClaims{"sub": "user123", "iss": "test"}

	zhtest.AssertTrue(t, hasClaim(claims, "sub"))
	zhtest.AssertTrue(t, hasClaim(claims, "iss"))
	zhtest.AssertFalse(t, hasClaim(claims, "aud"))
}

func TestGetStringClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			zhtest.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestGetMapClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			zhtest.AssertEqual(t, tt.expected, got)
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
			zhtest.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestGetJWTClaims(t *testing.T) {
	claims := map[string]any{"sub": "user123"}
	ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)

	jwt := GetClaims(req)
	zhtest.AssertNotNil(t, jwt.Raw())

	// Compare by checking we can access the same data
	wrappedClaims, ok := jwt.Raw().(map[string]any)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, "user123", wrappedClaims["sub"])

	// Test direct accessor
	zhtest.AssertEqual(t, "user123", jwt.Subject())
}

func TestClaimsWrapper_Subject(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			jwt := Claims{claims: tt.claims}
			got := jwt.Subject()
			zhtest.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestClaimsWrapper_Issuer(t *testing.T) {
	claims := map[string]any{"iss": "my-issuer"}
	jwt := Claims{claims: claims}

	zhtest.AssertEqual(t, "my-issuer", jwt.Issuer())
}

func TestClaimsWrapper_Audience(t *testing.T) {
	claims := map[string]any{"aud": "my-audience"}
	jwt := Claims{claims: claims}

	zhtest.AssertTrue(t, slicesEqual(jwt.Audience(), []string{"my-audience"}))
}

func TestClaimsWrapper_Audience_Array(t *testing.T) {
	claims := map[string]any{"aud": []string{"aud1", "aud2"}}
	jwt := Claims{claims: claims}

	zhtest.AssertTrue(t, slicesEqual(jwt.Audience(), []string{"aud1", "aud2"}))
}

func TestClaimsWrapper_HasAudience(t *testing.T) {
	claims := map[string]any{"aud": []string{"aud1", "aud2"}}
	jwt := Claims{claims: claims}

	zhtest.AssertTrue(t, jwt.HasAudience("aud1"))
	zhtest.AssertTrue(t, jwt.HasAudience("aud2"))
	zhtest.AssertFalse(t, jwt.HasAudience("aud3"))
}

func TestClaimsWrapper_JTI(t *testing.T) {
	claims := map[string]any{"jti": "token-id-123"}
	jwt := Claims{claims: claims}

	zhtest.AssertEqual(t, "token-id-123", jwt.JTI())
}

func TestClaimsWrapper_Expiration(t *testing.T) {
	now := time.Now()
	expUnix := now.Add(time.Hour).Unix()

	tests := []struct {
		name     string
		claims   JWTClaims
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
			jwt := Claims{claims: tt.claims}
			got := jwt.Expiration()
			zhtest.AssertTrue(t, got.Equal(tt.expected))
		})
	}
}

func TestClaimsWrapper_Scopes(t *testing.T) {
	tests := []struct {
		name     string
		claims   JWTClaims
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
			jwt := Claims{claims: tt.claims}
			got := jwt.Scopes()
			zhtest.AssertEqual(t, len(tt.expected), len(got))
			for i := range got {
				zhtest.AssertEqual(t, tt.expected[i], got[i])
			}
		})
	}
}

func TestClaimsWrapper_HasScope(t *testing.T) {
	claims := map[string]any{"scope": "read write admin"}
	jwt := Claims{claims: claims}

	zhtest.AssertTrue(t, jwt.HasScope("read"))
	zhtest.AssertTrue(t, jwt.HasScope("write"))
	zhtest.AssertTrue(t, jwt.HasScope("admin"))
	zhtest.AssertFalse(t, jwt.HasScope("delete"))
}

func TestClaimsWrapper_HasScope_NoClaims(t *testing.T) {
	jwt := Claims{claims: nil}
	zhtest.AssertFalse(t, jwt.HasScope("read"))
}

func TestGetJWTToken_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	token := GetToken(req)
	zhtest.AssertEqual(t, "", token)
}

func TestGetJWTError_NotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	err := GetError(req)
	zhtest.AssertNil(t, err)
}

func TestGetJWTError_NotJWTError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ErrorContextKey, errors.New("some error"))
	err := GetError(req.WithContext(ctx))
	zhtest.AssertNil(t, err)
}

func TestExpiration_NonMapClaims(t *testing.T) {
	// Test with non-map claims type
	jwt := Claims{claims: "not a map"}
	exp := jwt.Expiration()
	zhtest.AssertTrue(t, exp.IsZero())
}

func TestExpiration_Int64(t *testing.T) {
	claims := map[string]any{"exp": int64(time.Now().Add(time.Hour).Unix())}
	jwt := Claims{claims: claims}
	exp := jwt.Expiration()
	zhtest.AssertFalse(t, exp.IsZero())
}

func TestScopes_NonMapClaims(t *testing.T) {
	jwt := Claims{claims: "not a map"}
	scopes := jwt.Scopes()
	zhtest.AssertNil(t, scopes)
}

func TestScopes_EmptyString(t *testing.T) {
	jwt := Claims{claims: map[string]any{"scope": ""}}
	scopes := jwt.Scopes()
	zhtest.AssertEqual(t, 0, len(scopes))
}

func TestDefaultJWTErrorHandler_NoError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Call handler without setting error in context
	defaultJWTErrorHandler(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestExtractBearerToken_NoBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Basic abc123")

	token := extractBearerToken(req)
	zhtest.AssertEqual(t, "", token)
}

func TestExtractBearerToken_EmptyAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	token := extractBearerToken(req)
	zhtest.AssertEqual(t, "", token)
}

func TestDeepCopyMap_Nil(t *testing.T) {
	result := deepCopyMap(nil)
	zhtest.AssertNil(t, result)
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
	zhtest.AssertEqual(t, "alice", user["name"])
	tags := user["tags"].([]any)
	zhtest.AssertEqual(t, "admin", tags[0])
}

func TestDeepCopySlice_Nil(t *testing.T) {
	result := deepCopySlice(nil)
	zhtest.AssertNil(t, result)
}

func TestAddExpirationToClaims_NonMap(t *testing.T) {
	claims := "not a map"
	result := addExpirationToClaims(claims, time.Hour)
	zhtest.AssertEqual(t, claims, result)
}

func TestAddTypeToClaims_NonMap(t *testing.T) {
	claims := "not a map"
	result := addTypeToClaims(claims, "refresh")
	zhtest.AssertEqual(t, claims, result)
}

func TestGenerateAccessToken_DefaultTTL(t *testing.T) {
	store := &mockTokenStore{
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			return "token", nil
		},
	}
	cfg := Config{
		Store:          store,
		AccessTokenTTL: 0, // Should use default
	}
	claims := map[string]any{"sub": "user123"}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	token, err := GenerateAccessToken(req, claims, cfg)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, "token", token)
}

func TestRefreshTokenHandler_StoreError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
			}, nil
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			if tokenType == AccessToken {
				return "", errors.New("generate failed")
			}
			return "refresh", nil
		},
	}
	cfg := Config{
		Store: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusInternalServerError, rr.Code)
}

func TestLogoutTokenHandler_AlreadyRevoked(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
			}, nil
		},
		isRevoked: true,
	}
	cfg := Config{
		Store: store,
	}
	handler := LogoutTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)
}

func TestJWTAuth_RevokedToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
		isRevoked: true,
	}

	middleware := New(Config{
		Store: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer revoked-token")
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer revoked-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

func TestJWTAuth_RefreshTokenAsAccessToken(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
			}, nil
		},
	}

	middleware := New(Config{
		Store: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer refresh-token")
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusUnauthorized, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer refresh-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

func TestJWTAuth_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			if token == "valid-token" {
				return map[string]any{"sub": "user123"}, nil
			}
			return nil, errors.New("invalid token")
		},
	}
	mw := New(Config{
		Store: store,
	})

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, metrics.Config{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})
	wrapped := metricsMw(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// Request without token
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr1, req1)
	zhtest.AssertEqual(t, http.StatusUnauthorized, rr1.Code)

	// Request with valid token
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)
	zhtest.AssertEqual(t, http.StatusOK, rr2.Code)

	// Check metrics
	families := reg.Gather()
	var counter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "jwt_auth_requests_total" {
			counter = &f
			break
		}
	}
	zhtest.AssertNotNil(t, counter)

	// Should have metrics for both valid and invalid
	results := make(map[string]int)
	for _, m := range counter.Metrics {
		results[m.Labels["result"]]++
	}
	zhtest.AssertEqual(t, 1, results["missing"])
	zhtest.AssertEqual(t, 1, results["valid"])
}

// Test for IsRevoked error path in New middleware
func TestJWTAuth_IsRevokedCheckError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
		isRevokedFunc: func(ctx context.Context, claims map[string]any) (bool, error) {
			return false, errors.New("database connection failed")
		},
	}

	middleware := New(Config{
		Store: store,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test JSON response
	req := httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	req.Header.Set(httpx.HeaderAccept, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusInternalServerError, rr.Code)

	var errResp AuthError
	zhtest.AssertNoError(t, json.Unmarshal(rr.Body.Bytes(), &errResp))
	zhtest.AssertContains(t, errResp.Title, "Token Revocation Check Failed")

	// Test plain text response
	req = httptest.NewRequest(http.MethodGet, "/api/protected", nil)
	req.Header.Set(httpx.HeaderAuthorization, "Bearer valid-token")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	zhtest.AssertContains(t, rr.Header().Get(httpx.HeaderContentType), "text/plain")
}

// Test for IsRevoked error path in RefreshTokenHandler
func TestRefreshTokenHandler_IsRevokedCheckError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
			}, nil
		},
		isRevokedFunc: func(ctx context.Context, claims map[string]any) (bool, error) {
			return false, errors.New("database connection failed")
		},
	}
	cfg := Config{
		Store: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusInternalServerError, rr.Code)
}

// Test for getStringClaim with []any slice containing non-string elements
func TestGetStringClaim_AnySliceWithNonString(t *testing.T) {
	claims := map[string]any{
		"aud": []any{123, "aud2"}, // First element is not a string
	}

	result := getStringClaim(claims, "aud")
	zhtest.AssertEqual(t, "", result)

	// Test with empty []any slice
	claims2 := map[string]any{
		"aud": []any{},
	}

	result2 := getStringClaim(claims2, "aud")
	zhtest.AssertEqual(t, "", result2)
}

// Test for getStringClaim with reflection path for non-map claims
func TestGetStringClaim_ReflectionPath(t *testing.T) {
	// Use a custom map type that requires reflection
	claims := customMapClaims{
		"sub": "user123",
	}

	result := getStringClaim(claims, "sub")
	zhtest.AssertEqual(t, "user123", result)

	// Test missing key via reflection
	result2 := getStringClaim(claims, "missing")
	zhtest.AssertEqual(t, "", result2)
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
	zhtest.AssertEqual(t, "simple", copied[0])
	nested := copied[1].([]any)
	zhtest.AssertEqual(t, "nested1", nested[0])
	m := copied[2].(map[string]any)
	zhtest.AssertEqual(t, "value", m["key"])
}

// Test for addExpirationToClaims with HS256Claims
func TestAddExpirationToClaims_HS256Claims(t *testing.T) {
	ttl := 15 * time.Minute
	claims := HS256Claims{"sub": "user123"}

	result := addExpirationToClaims(claims, ttl)

	resultMap, ok := result.(HS256Claims)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, "user123", resultMap["sub"])

	_, exists := resultMap["exp"]
	zhtest.AssertTrue(t, exists)

	// Original should not be modified
	_, exists = claims["exp"]
	zhtest.AssertFalse(t, exists)
}

// Test for addTypeToClaims with HS256Claims
func TestAddTypeToClaims_HS256Claims(t *testing.T) {
	claims := HS256Claims{"sub": "user123"}

	result := addTypeToClaims(claims, "refresh")

	resultMap, ok := result.(HS256Claims)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, "refresh", resultMap["type"])

	// Original should not be modified
	_, exists := claims["type"]
	zhtest.AssertFalse(t, exists)
}

// Test for RefreshTokenHandler GenerateRefreshToken error
func TestRefreshTokenHandler_GenerateRefreshTokenError(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{
				"sub":  "user123",
				"type": TokenTypeRefresh,
			}, nil
		},
		generateFunc: func(ctx context.Context, claims JWTClaims, tokenType TokenType, ttl time.Duration) (string, error) {
			if tokenType == RefreshToken {
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
	cfg := Config{
		Store: store,
	}
	handler := RefreshTokenHandler(cfg)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(body))
	req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationJSON)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusInternalServerError, rr.Code)
}

func TestJWTAuth_IncludedPaths(t *testing.T) {
	store := &mockTokenStore{
		validateFunc: func(ctx context.Context, token string) (JWTClaims, error) {
			return map[string]any{"sub": "user123"}, nil
		},
	}

	mw := New(Config{
		Store:         store,
		IncludedPaths: []string{"/api/admin", "/api/private/"},
	})

	tests := []struct {
		name       string
		path       string
		token      string
		wantStatus int
	}{
		{"allowed path with token", "/api/admin", "valid-token", http.StatusOK},
		{"allowed path without token", "/api/admin", "", http.StatusUnauthorized},
		{"allowed prefix path with token", "/api/private/data", "valid-token", http.StatusOK},
		{"allowed prefix path without token", "/api/private/data", "", http.StatusUnauthorized},
		{"non-allowed path with token", "/public", "valid-token", http.StatusOK},
		{"non-allowed path without token", "/public", "", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.token != "" {
				req.Header.Set(httpx.HeaderAuthorization, "Bearer "+tt.token)
			}
			rr := httptest.NewRecorder()
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			zhtest.AssertEqual(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestJWTAuth_BothExcludedAndIncludedPathsPanics(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			Store:         &mockTokenStore{},
			ExcludedPaths: []string{"/health"},
			IncludedPaths: []string{"/api"},
		})
	})
}

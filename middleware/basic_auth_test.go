package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func createAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func testMiddleware(t *testing.T, middleware func(http.Handler) http.Handler, req *http.Request, expectAuth bool, expectedStatus int) {
	t.Helper()
	rr := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if called != expectAuth {
		t.Errorf("expected auth %v, got %v", expectAuth, called)
	}
	if rr.Code != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, rr.Code)
	}
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name           string
		middleware     func(http.Handler) http.Handler
		path           string
		authHeader     string
		expectAuth     bool
		expectedStatus int
		checkWWWAuth   bool
		expectedRealm  string
	}{
		{
			name:           "no config denies all",
			middleware:     BasicAuth(),
			path:           "/test",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
			checkWWWAuth:   true,
			expectedRealm:  `Basic realm="Restricted"`,
		},
		{
			name:           "valid credentials",
			middleware:     BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"})),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "secret"),
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid password",
			middleware:     BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"})),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "wrong"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unknown user",
			middleware:     BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"})),
			path:           "/test",
			authHeader:     createAuthHeader("hacker", "password"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "validator accepts valid credentials",
			middleware:     BasicAuth(config.WithBasicAuthValidator(func(u, p string) bool { return u == "test" && p == "pass" })),
			path:           "/test",
			authHeader:     createAuthHeader("test", "pass"),
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "validator rejects invalid credentials",
			middleware:     BasicAuth(config.WithBasicAuthValidator(func(u, p string) bool { return u == "test" && p == "pass" })),
			path:           "/test",
			authHeader:     createAuthHeader("wrong", "pass"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "exempt path allows access",
			middleware:     BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}), config.WithBasicAuthExemptPaths([]string{"/health"})),
			path:           "/health",
			authHeader:     "",
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-exempt path requires auth",
			middleware:     BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}), config.WithBasicAuthExemptPaths([]string{"/health"})),
			path:           "/admin",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "custom realm",
			middleware:     BasicAuth(config.WithBasicAuthRealm("Custom Realm"), config.WithBasicAuthCredentials(map[string]string{"admin": "secret"})),
			path:           "/test",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
			checkWWWAuth:   true,
			expectedRealm:  `Basic realm="Custom Realm"`,
		},
		{
			name:           "empty realm falls back to default",
			middleware:     BasicAuth(config.WithBasicAuthRealm(""), config.WithBasicAuthCredentials(map[string]string{"admin": "secret"})),
			path:           "/test",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
			checkWWWAuth:   true,
			expectedRealm:  `Basic realm="Restricted"`,
		},
		{
			name:           "validator takes precedence over credentials",
			middleware:     BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}), config.WithBasicAuthValidator(func(u, p string) bool { return u == "validator" && p == "test" })),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "secret"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()
			called := false
			tt.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rr, req)

			if called != tt.expectAuth {
				t.Errorf("expected auth %v, got %v", tt.expectAuth, called)
			}
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.checkWWWAuth {
				if auth := rr.Header().Get("WWW-Authenticate"); auth != tt.expectedRealm {
					t.Errorf("expected %s, got %s", tt.expectedRealm, auth)
				}
			}
		})
	}
}

func TestBasicAuthMalformedHeaders(t *testing.T) {
	middleware := BasicAuth(config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}))
	malformedHeaders := []string{"", "InvalidFormat", "Bearer token123", "Basic invalidbase64!!!"}

	for _, header := range malformedHeaders {
		t.Run("malformed_"+header, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			testMiddleware(t, middleware, req, false, http.StatusUnauthorized)
		})
	}
}

func TestBasicAuthExemptPaths(t *testing.T) {
	middleware := BasicAuth(
		config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}),
		config.WithBasicAuthExemptPaths([]string{"/health", "/metrics", "/api/public/"}),
	)

	pathTests := []struct {
		path   string
		exempt bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/api/public/", true},
		{"/api/public/users", true},
		{"/api/private", false},
		{"/admin", false},
	}

	for _, tt := range pathTests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			expectedStatus := http.StatusUnauthorized
			if tt.exempt {
				expectedStatus = http.StatusOK
			}
			testMiddleware(t, middleware, req, tt.exempt, expectedStatus)
		})
	}
}

func TestBasicAuthNoAuthConfigured(t *testing.T) {
	middleware := BasicAuth(config.WithBasicAuthRealm("Test Realm"))

	req := httptest.NewRequest("GET", "/test", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("user:password"))
	req.Header.Set("Authorization", "Basic "+auth)
	rr := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	if called {
		t.Error("handler should not be called when no auth configured")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestBasicAuthNilExemptPathsFallback(t *testing.T) {
	middleware := BasicAuth(
		config.WithBasicAuthCredentials(map[string]string{"admin": "secret"}),
		config.WithBasicAuthExemptPaths(nil),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	if called {
		t.Error("handler should not be called without auth")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestBasicAuthFailedFunction(t *testing.T) {
	rr := httptest.NewRecorder()
	basicAuthFailed(rr, "Test Realm")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if auth := rr.Header().Get("WWW-Authenticate"); auth != `Basic realm="Test Realm"` {
		t.Errorf("expected Test Realm, got %s", auth)
	}
}

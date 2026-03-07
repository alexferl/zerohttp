package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func createAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func testMiddleware(t *testing.T, middleware func(http.Handler) http.Handler, req *http.Request, expectAuth bool, expectedStatus int) {
	t.Helper()
	w := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
		rw.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if called != expectAuth {
		t.Errorf("expected auth %v, got %v", expectAuth, called)
	}
	zhtest.AssertWith(t, w).Status(expectedStatus)
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
			name: "valid credentials",
			middleware: BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "secret"),
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid password",
			middleware: BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "wrong"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "unknown user",
			middleware: BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     createAuthHeader("hacker", "password"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "validator accepts valid credentials",
			middleware: BasicAuth(config.BasicAuthConfig{
				Validator: func(u, p string) bool { return u == "test" && p == "pass" },
			}),
			path:           "/test",
			authHeader:     createAuthHeader("test", "pass"),
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "validator rejects invalid credentials",
			middleware: BasicAuth(config.BasicAuthConfig{
				Validator: func(u, p string) bool { return u == "test" && p == "pass" },
			}),
			path:           "/test",
			authHeader:     createAuthHeader("wrong", "pass"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "exempt path allows access",
			middleware: BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{"admin": "secret"},
				ExemptPaths: []string{"/health"},
			}),
			path:           "/health",
			authHeader:     "",
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-exempt path requires auth",
			middleware: BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{"admin": "secret"},
				ExemptPaths: []string{"/health"},
			}),
			path:           "/admin",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "custom realm",
			middleware: BasicAuth(config.BasicAuthConfig{
				Realm:       "Custom Realm",
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
			checkWWWAuth:   true,
			expectedRealm:  `Basic realm="Custom Realm"`,
		},
		{
			name: "empty realm falls back to default",
			middleware: BasicAuth(config.BasicAuthConfig{
				Realm:       "",
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
			checkWWWAuth:   true,
			expectedRealm:  `Basic realm="Restricted"`,
		},
		{
			name: "validator takes precedence over credentials",
			middleware: BasicAuth(config.BasicAuthConfig{
				Credentials: map[string]string{"admin": "secret"},
				Validator:   func(u, p string) bool { return u == "validator" && p == "test" },
			}),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "secret"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			called := false
			tt.middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				called = true
				rw.WriteHeader(http.StatusOK)
			})).ServeHTTP(w, req)

			if called != tt.expectAuth {
				t.Errorf("expected auth %v, got %v", tt.expectAuth, called)
			}
			zhtest.AssertWith(t, w).Status(tt.expectedStatus)
			if tt.checkWWWAuth {
				zhtest.AssertWith(t, w).Header("WWW-Authenticate", tt.expectedRealm)
			}
		})
	}
}

func TestBasicAuthMalformedHeaders(t *testing.T) {
	middleware := BasicAuth(config.BasicAuthConfig{
		Credentials: map[string]string{"admin": "secret"},
	})
	malformedHeaders := []string{"", "InvalidFormat", "Bearer token123", "Basic invalidbase64!!!"}

	for _, header := range malformedHeaders {
		t.Run("malformed_"+header, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			testMiddleware(t, middleware, req, false, http.StatusUnauthorized)
		})
	}
}

func TestBasicAuthExemptPaths(t *testing.T) {
	middleware := BasicAuth(config.BasicAuthConfig{
		Credentials: map[string]string{"admin": "secret"},
		ExemptPaths: []string{"/health", "/metrics", "/api/public/"},
	})

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
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			expectedStatus := http.StatusUnauthorized
			if tt.exempt {
				expectedStatus = http.StatusOK
			}
			testMiddleware(t, middleware, req, tt.exempt, expectedStatus)
		})
	}
}

func TestBasicAuthNoAuthConfigured(t *testing.T) {
	middleware := BasicAuth(config.BasicAuthConfig{
		Realm: "Test Realm",
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("user:password"))
	req.Header.Set("Authorization", "Basic "+auth)
	w := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called {
		t.Error("handler should not be called when no auth configured")
	}
	zhtest.AssertWith(t, w).Status(http.StatusUnauthorized)
}

func TestBasicAuthNilExemptPathsFallback(t *testing.T) {
	middleware := BasicAuth(config.BasicAuthConfig{
		Credentials: map[string]string{"admin": "secret"},
		ExemptPaths: nil,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called {
		t.Error("handler should not be called without auth")
	}
	zhtest.AssertWith(t, w).Status(http.StatusUnauthorized)
}

func TestBasicAuthFailedFunction(t *testing.T) {
	w := httptest.NewRecorder()
	basicAuthFailed(w, "Test Realm")

	zhtest.AssertWith(t, w).
		Status(http.StatusUnauthorized).
		Header("WWW-Authenticate", `Basic realm="Test Realm"`)
}

package basicauth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/metrics"
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

	zhtest.AssertEqual(t, expectAuth, called)
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
			middleware:     New(),
			path:           "/test",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
			checkWWWAuth:   true,
			expectedRealm:  `Basic realm="Restricted"`,
		},
		{
			name: "valid credentials",
			middleware: New(Config{
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "secret"),
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid password",
			middleware: New(Config{
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     createAuthHeader("admin", "wrong"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "unknown user",
			middleware: New(Config{
				Credentials: map[string]string{"admin": "secret"},
			}),
			path:           "/test",
			authHeader:     createAuthHeader("hacker", "password"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "validator accepts valid credentials",
			middleware: New(Config{
				Validator: func(u, p string) bool { return u == "test" && p == "pass" },
			}),
			path:           "/test",
			authHeader:     createAuthHeader("test", "pass"),
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "validator rejects invalid credentials",
			middleware: New(Config{
				Validator: func(u, p string) bool { return u == "test" && p == "pass" },
			}),
			path:           "/test",
			authHeader:     createAuthHeader("wrong", "pass"),
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "excluded path allows access",
			middleware: New(Config{
				Credentials:   map[string]string{"admin": "secret"},
				ExcludedPaths: []string{"/health"},
			}),
			path:           "/health",
			authHeader:     "",
			expectAuth:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "non-excluded path requires auth",
			middleware: New(Config{
				Credentials:   map[string]string{"admin": "secret"},
				ExcludedPaths: []string{"/health"},
			}),
			path:           "/admin",
			authHeader:     "",
			expectAuth:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "custom realm",
			middleware: New(Config{
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
			middleware: New(Config{
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
			middleware: New(Config{
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
				req.Header.Set(httpx.HeaderAuthorization, tt.authHeader)
			}
			w := httptest.NewRecorder()
			called := false
			tt.middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				called = true
				rw.WriteHeader(http.StatusOK)
			})).ServeHTTP(w, req)

			zhtest.AssertEqual(t, tt.expectAuth, called)
			zhtest.AssertWith(t, w).Status(tt.expectedStatus)
			if tt.checkWWWAuth {
				zhtest.AssertWith(t, w).Header(httpx.HeaderWWWAuthenticate, tt.expectedRealm)
			}
		})
	}
}

func TestBasicAuthMalformedHeaders(t *testing.T) {
	middleware := New(Config{
		Credentials: map[string]string{"admin": "secret"},
	})
	malformedHeaders := []string{"", "InvalidFormat", "Bearer token123", "Basic invalidbase64!!!"}

	for _, header := range malformedHeaders {
		t.Run("malformed_"+header, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if header != "" {
				req.Header.Set(httpx.HeaderAuthorization, header)
			}
			testMiddleware(t, middleware, req, false, http.StatusUnauthorized)
		})
	}
}

func TestBasicAuthExcludedPaths(t *testing.T) {
	middleware := New(Config{
		Credentials:   map[string]string{"admin": "secret"},
		ExcludedPaths: []string{"/health", "/metrics", "/api/public/"},
	})

	pathTests := []struct {
		path     string
		excluded bool
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
			if tt.excluded {
				expectedStatus = http.StatusOK
			}
			testMiddleware(t, middleware, req, tt.excluded, expectedStatus)
		})
	}
}

func TestBasicAuthIncludedPaths(t *testing.T) {
	middleware := New(Config{
		Credentials:   map[string]string{"admin": "secret"},
		IncludedPaths: []string{"/admin", "/api/private/"},
	})

	pathTests := []struct {
		path       string
		shouldAuth bool
	}{
		{"/admin", true},
		{"/api/private/", true},
		{"/api/private/data", true},
		{"/health", false},
		{"/metrics", false},
		{"/public", false},
	}

	for _, tt := range pathTests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			expectedStatus := http.StatusOK
			if tt.shouldAuth {
				expectedStatus = http.StatusUnauthorized
			}
			testMiddleware(t, middleware, req, !tt.shouldAuth, expectedStatus)
		})
	}
}

func TestBasicAuthIncludedPathsWithAuth(t *testing.T) {
	middleware := New(Config{
		Credentials:   map[string]string{"admin": "secret"},
		IncludedPaths: []string{"/admin"},
	})

	t.Run("allowed path with valid auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(httpx.HeaderAuthorization, createAuthHeader("admin", "secret"))
		testMiddleware(t, middleware, req, true, http.StatusOK)
	})

	t.Run("allowed path with invalid auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set(httpx.HeaderAuthorization, createAuthHeader("admin", "wrong"))
		testMiddleware(t, middleware, req, false, http.StatusUnauthorized)
	})

	t.Run("non-allowed path without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/public", nil)
		testMiddleware(t, middleware, req, true, http.StatusOK)
	})
}

func TestBasicAuthBothExcludedAndIncludedPathsPanics(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		_ = New(Config{
			Credentials:   map[string]string{"admin": "secret"},
			ExcludedPaths: []string{"/health"},
			IncludedPaths: []string{"/admin"},
		})
	})
}

func TestBasicAuthNoAuthConfigured(t *testing.T) {
	middleware := New(Config{
		Realm: "Test Realm",
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("user:password"))
	req.Header.Set(httpx.HeaderAuthorization, "Basic "+auth)
	w := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	zhtest.AssertFalse(t, called)
	zhtest.AssertWith(t, w).Status(http.StatusUnauthorized)
}

func TestBasicAuthNilExcludedPathsFallback(t *testing.T) {
	middleware := New(Config{
		Credentials:   map[string]string{"admin": "secret"},
		ExcludedPaths: nil,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	called := false
	middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	zhtest.AssertFalse(t, called)
	zhtest.AssertWith(t, w).Status(http.StatusUnauthorized)
}

func TestBasicAuthFailedFunction(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	basicAuthFailed(w, req, "Test Realm")

	zhtest.AssertWith(t, w).
		Status(http.StatusUnauthorized).
		Header(httpx.HeaderWWWAuthenticate, `Basic realm="Test Realm"`)
}

func TestBasicAuth_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	mw := New(Config{
		Credentials: map[string]string{"admin": "secret"},
	})

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, metrics.Config{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})
	wrapped := metricsMw(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	// Test missing auth
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()
	wrapped.ServeHTTP(w1, req1)
	zhtest.AssertEqual(t, http.StatusUnauthorized, w1.Code)

	// Test valid auth
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	auth := base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	req2.Header.Set(httpx.HeaderAuthorization, "Basic "+auth)
	w2 := httptest.NewRecorder()
	wrapped.ServeHTTP(w2, req2)
	zhtest.AssertEqual(t, http.StatusOK, w2.Code)

	// Check metrics
	families := reg.Gather()
	var counter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "basic_auth_requests_total" {
			counter = &f
			break
		}
	}
	zhtest.AssertNotNil(t, counter)

	// Should have metrics for both valid and missing
	results := make(map[string]int)
	for _, m := range counter.Metrics {
		results[m.Labels["result"]]++
	}
	zhtest.AssertEqual(t, 1, results["missing"])
	zhtest.AssertEqual(t, 1, results["valid"])
}

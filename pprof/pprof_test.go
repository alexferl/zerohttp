package pprof

import (
	"net/http"
	"net/http/httptest"
	"testing"

	zh "github.com/alexferl/zerohttp"
)

// setupPProf creates a test server with pprof endpoints and returns the server and pprof instance.
// By default, auth is disabled for tests. Use setupPProfWithAuth to test auth scenarios.
func setupPProf(t *testing.T, cfg *Config) (*zh.Server, *PProf) {
	t.Helper()
	app := zh.New()

	var c Config
	if cfg != nil {
		c = *cfg
		// Disable auth for tests unless explicitly set
		if cfg.Auth == nil {
			c.Auth = &AuthConfig{}
		}
	} else {
		c = DefaultConfig
		c.Auth = &AuthConfig{} // disable auth by default for tests
	}

	pp := New(app, c)
	return app, pp
}

// setupPProfWithAuth creates a test server with pprof endpoints and specific auth credentials.
func setupPProfWithAuth(t *testing.T, username, password string) (*zh.Server, *PProf) {
	t.Helper()
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{
		Username: username,
		Password: password,
	}
	pp := New(app, cfg)
	return app, pp
}

// makeRequest makes an HTTP request to the test server and returns the response.
func makeRequest(t *testing.T, app *zh.Server, method, path string, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig.Prefix != "/debug/pprof" {
		t.Errorf("expected prefix '/debug/pprof', got '%s'", DefaultConfig.Prefix)
	}
	if !DefaultConfig.EnableIndex {
		t.Error("expected EnableIndex to be true")
	}
	if !DefaultConfig.EnableCmdline {
		t.Error("expected EnableCmdline to be true")
	}
	if !DefaultConfig.EnableProfile {
		t.Error("expected EnableProfile to be true")
	}
	if !DefaultConfig.EnableSymbol {
		t.Error("expected EnableSymbol to be true")
	}
	if !DefaultConfig.EnableTrace {
		t.Error("expected EnableTrace to be true")
	}
	if !DefaultConfig.EnableHeap {
		t.Error("expected EnableHeap to be true")
	}
	if !DefaultConfig.EnableGoroutine {
		t.Error("expected EnableGoroutine to be true")
	}
	if !DefaultConfig.EnableThreadCreate {
		t.Error("expected EnableThreadCreate to be true")
	}
	if !DefaultConfig.EnableBlock {
		t.Error("expected EnableBlock to be true")
	}
	if !DefaultConfig.EnableMutex {
		t.Error("expected EnableMutex to be true")
	}
	if DefaultConfig.Auth != nil {
		t.Error("expected Auth to be nil")
	}
}

func TestNewWithDefaultConfig(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewWithCustomPrefix(t *testing.T) {
	cfg := DefaultConfig
	cfg.Prefix = "/admin/pprof"
	app, _ := setupPProf(t, &cfg)

	// Test index endpoint at custom prefix
	rec := makeRequest(t, app, http.MethodGet, "/admin/pprof/", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Test that default prefix doesn't work
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d for unknown path, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestNewWithAuth(t *testing.T) {
	app, _ := setupPProfWithAuth(t, "admin", "secret")

	// Test without auth
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d without auth, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Test with wrong credentials
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", "wrong", "wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d with wrong credentials, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Test with correct credentials
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", "admin", "secret")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with correct credentials, got %d", http.StatusOK, rec.Code)
	}
}

func TestNewWithDisabledEndpoints(t *testing.T) {
	cfg := DefaultConfig
	cfg.EnableIndex = false
	cfg.EnableCmdline = false
	app, _ := setupPProf(t, &cfg)

	// Test disabled index endpoint
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d for disabled endpoint, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestCmdlineEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/cmdline", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSymbolEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	// Test GET
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/symbol", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for GET, got %d", http.StatusOK, rec.Code)
	}

	// Test POST
	rec = makeRequest(t, app, http.MethodPost, "/debug/pprof/symbol", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d for POST, got %d", http.StatusOK, rec.Code)
	}
}

func TestHeapEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/heap", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGoroutineEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/goroutine", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestThreadCreateEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/threadcreate", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestBlockEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/block", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMutexEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/mutex", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAllEndpointsWithAuth(t *testing.T) {
	app, pp := setupPProfWithAuth(t, "test", "test")

	if pp.Auth == nil {
		t.Fatal("expected Auth to be set")
	}
	if pp.Auth.Username != "test" {
		t.Errorf("expected username 'test', got '%s'", pp.Auth.Username)
	}
	if pp.Auth.Password != "test" {
		t.Errorf("expected password 'test', got '%s'", pp.Auth.Password)
	}

	endpoints := []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/symbol",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
		"/debug/pprof/threadcreate",
		"/debug/pprof/block",
		"/debug/pprof/mutex",
	}

	for _, endpoint := range endpoints {
		// Without auth
		rec := makeRequest(t, app, http.MethodGet, endpoint, "", "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("endpoint %s: expected status %d without auth, got %d", endpoint, http.StatusUnauthorized, rec.Code)
		}

		// With auth
		rec = makeRequest(t, app, http.MethodGet, endpoint, "test", "test")
		if rec.Code != http.StatusOK {
			t.Errorf("endpoint %s: expected status %d with auth, got %d", endpoint, http.StatusOK, rec.Code)
		}
	}
}

func TestAutoGeneratedAuth(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig // Auth is nil, so auto-generate
	pp := New(app, cfg)

	if pp.Auth == nil {
		t.Fatal("expected Auth to be auto-generated")
	}
	if pp.Auth.Username != "pprof" {
		t.Errorf("expected default username 'pprof', got '%s'", pp.Auth.Username)
	}
	if pp.Auth.Password == "" {
		t.Error("expected auto-generated password to not be empty")
	}

	// Test that auth is required
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d without auth, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Test with auto-generated credentials
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", pp.Auth.Username, pp.Auth.Password)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with correct credentials, got %d", http.StatusOK, rec.Code)
	}
}

func TestDisabledAuth(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // empty = disabled
	pp := New(app, cfg)

	if pp.Auth != nil {
		t.Error("expected Auth to be nil when disabled")
	}

	// Test without auth - should succeed
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with auth disabled, got %d", http.StatusOK, rec.Code)
	}
}

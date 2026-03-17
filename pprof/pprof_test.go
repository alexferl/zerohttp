package pprof

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
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
// Requests come from localhost (127.0.0.1) by default to pass IP allowlist checks.
func makeRequest(t *testing.T, app *zh.Server, method, path string, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	// Set RemoteAddr to localhost to pass default IP allowlist
	req.RemoteAddr = "127.0.0.1:1234"
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

// makeRequestFromIP makes an HTTP request from a specific IP address.
// For IPv6 addresses, use the bracket notation like "[::1]".
func makeRequestFromIP(t *testing.T, app *zh.Server, method, path, clientIP, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	// Format RemoteAddr properly for IPv4 and IPv6
	if strings.Contains(clientIP, ":") {
		// IPv6 - needs brackets
		req.RemoteAddr = "[" + clientIP + "]:1234"
	} else {
		// IPv4
		req.RemoteAddr = clientIP + ":1234"
	}
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func TestNewWithNoConfig(t *testing.T) {
	// Test calling New without any config (uses defaults)
	app := zh.New()
	pp := New(app)

	if pp == nil {
		t.Fatal("expected PProf instance, got nil")
	}

	if pp.Config.Prefix != "/debug/pprof" {
		t.Errorf("expected prefix '/debug/pprof', got '%s'", pp.Config.Prefix)
	}

	if pp.Auth == nil {
		t.Fatal("expected Auth to be auto-generated")
	}

	if pp.Auth.Username != "pprof" {
		t.Errorf("expected default username 'pprof', got '%s'", pp.Auth.Username)
	}
}

func TestNewWithPartialConfig(t *testing.T) {
	// Test partial config merging with defaults
	app := zh.New()

	// Only override prefix, rest should use defaults
	pp := New(app, Config{
		Prefix: "/custom/pprof",
	})

	if pp.Config.Prefix != "/custom/pprof" {
		t.Errorf("expected prefix '/custom/pprof', got '%s'", pp.Config.Prefix)
	}

	// Auth should still be auto-generated (nil in partial config = use default behavior)
	if pp.Auth == nil {
		t.Fatal("expected Auth to be auto-generated")
	}

	// Other fields should use defaults
	if !*pp.Config.EnableIndex {
		t.Error("expected EnableIndex to be true from defaults")
	}

	if !*pp.Config.EnableHeap {
		t.Error("expected EnableHeap to be true from defaults")
	}

	if len(pp.Config.AllowedIPs) != 2 {
		t.Errorf("expected 2 allowed IPs from defaults, got %d", len(pp.Config.AllowedIPs))
	}
}

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig.Prefix != "/debug/pprof" {
		t.Errorf("expected prefix '/debug/pprof', got '%s'", DefaultConfig.Prefix)
	}
	if !config.BoolOrDefault(DefaultConfig.EnableIndex, false) {
		t.Error("expected EnableIndex to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableCmdline, false) {
		t.Error("expected EnableCmdline to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableProfile, false) {
		t.Error("expected EnableProfile to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableSymbol, false) {
		t.Error("expected EnableSymbol to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableTrace, false) {
		t.Error("expected EnableTrace to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableHeap, false) {
		t.Error("expected EnableHeap to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableGoroutine, false) {
		t.Error("expected EnableGoroutine to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableThreadCreate, false) {
		t.Error("expected EnableThreadCreate to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableBlock, false) {
		t.Error("expected EnableBlock to be true")
	}
	if !config.BoolOrDefault(DefaultConfig.EnableMutex, false) {
		t.Error("expected EnableMutex to be true")
	}
	if DefaultConfig.Auth != nil {
		t.Error("expected Auth to be nil")
	}
	if len(DefaultConfig.AllowedIPs) != 2 {
		t.Errorf("expected AllowedIPs to have 2 entries (localhost only), got %d", len(DefaultConfig.AllowedIPs))
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
	cfg.EnableIndex = config.Bool(false)
	cfg.EnableCmdline = config.Bool(false)
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

	// Test without auth - should succeed (from localhost)
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with auth disabled, got %d", http.StatusOK, rec.Code)
	}
}

func TestIPAllowlistDefault(t *testing.T) {
	// Default config should only allow localhost
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // disable auth for this test
	New(app, cfg)

	// Request from localhost should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "127.0.0.1", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d from localhost, got %d", http.StatusOK, rec.Code)
	}

	// Request from external IP should be forbidden
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.1.100", "", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d from external IP, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestIPAllowlistCustom(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // disable auth for this test
	cfg.AllowedIPs = []string{"192.168.1.0/24", "10.0.0.100"}
	New(app, cfg)

	// Request from allowed CIDR should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.1.50", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d from allowed CIDR, got %d", http.StatusOK, rec.Code)
	}

	// Request from allowed single IP should succeed
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "10.0.0.100", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d from allowed IP, got %d", http.StatusOK, rec.Code)
	}

	// Request from disallowed IP should be forbidden
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.2.1", "", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d from disallowed IP, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestIPAllowlistDisabled(t *testing.T) {
	// Empty slice disables IP checking
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{}    // disable auth for this test
	cfg.AllowedIPs = []string{} // empty = allow any IP
	New(app, cfg)

	// Request from any IP should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "8.8.8.8", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d when IP allowlist disabled, got %d", http.StatusOK, rec.Code)
	}
}

func TestIPAllowlistWithAuth(t *testing.T) {
	// Test that IP check happens before auth
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{Username: "admin", Password: "secret"}
	cfg.AllowedIPs = []string{"127.0.0.1/32"}
	New(app, cfg)

	// Request from disallowed IP should get 403 before auth check
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.1.1", "admin", "secret")
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d from disallowed IP, got %d", http.StatusForbidden, rec.Code)
	}

	// Request from allowed IP with wrong auth should get 401
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "127.0.0.1", "wrong", "wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d with wrong auth, got %d", http.StatusUnauthorized, rec.Code)
	}

	// Request from allowed IP with correct auth should succeed
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "127.0.0.1", "admin", "secret")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d with correct auth, got %d", http.StatusOK, rec.Code)
	}
}

func TestIPAllowlistIPv6(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // disable auth for this test
	cfg.AllowedIPs = []string{"::1/128", "2001:db8::/32"}
	New(app, cfg)

	// Request from localhost IPv6 should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "::1", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d from ::1, got %d", http.StatusOK, rec.Code)
	}

	// Request from allowed IPv6 CIDR should succeed
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "2001:db8::1", "", "")
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d from allowed IPv6, got %d", http.StatusOK, rec.Code)
	}

	// Request from disallowed IPv6 should be forbidden
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "2001:db9::1", "", "")
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d from disallowed IPv6, got %d", http.StatusForbidden, rec.Code)
	}
}

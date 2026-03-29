package pprof

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
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

	zhtest.AssertNotNil(t, pp)
	zhtest.AssertEqual(t, "/debug/pprof", pp.Config.Prefix)
	zhtest.AssertNotNil(t, pp.Auth)
	zhtest.AssertEqual(t, "pprof", pp.Auth.Username)
}

func TestNewWithPartialConfig(t *testing.T) {
	// Test partial config merging with defaults
	app := zh.New()

	// Only override prefix, rest should use defaults
	pp := New(app, Config{
		Prefix: "/custom/pprof",
	})

	zhtest.AssertEqual(t, "/custom/pprof", pp.Config.Prefix)
	// Auth should still be auto-generated (nil in partial config = use default behavior)
	zhtest.AssertNotNil(t, pp.Auth)
	// Other fields should use defaults
	zhtest.AssertTrue(t, *pp.Config.EnableIndex)
	zhtest.AssertTrue(t, *pp.Config.EnableHeap)
	zhtest.AssertEqual(t, 2, len(pp.Config.AllowedIPs))
}

func TestDefaultConfig(t *testing.T) {
	zhtest.AssertEqual(t, "/debug/pprof", DefaultConfig.Prefix)
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableIndex, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableCmdline, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableProfile, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableSymbol, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableTrace, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableHeap, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableGoroutine, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableThreadCreate, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableBlock, false))
	zhtest.AssertTrue(t, config.BoolOrDefault(DefaultConfig.EnableMutex, false))
	zhtest.AssertNil(t, DefaultConfig.Auth)
	zhtest.AssertEqual(t, 2, len(DefaultConfig.AllowedIPs))
}

func TestNewWithDefaultConfig(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestNewWithCustomPrefix(t *testing.T) {
	cfg := DefaultConfig
	cfg.Prefix = "/admin/pprof"
	app, _ := setupPProf(t, &cfg)

	// Test index endpoint at custom prefix
	rec := makeRequest(t, app, http.MethodGet, "/admin/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Test that default prefix doesn't work
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusNotFound, rec.Code)
}

func TestNewWithAuth(t *testing.T) {
	app, _ := setupPProfWithAuth(t, "admin", "secret")

	// Test without auth
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusUnauthorized, rec.Code)

	// Test with wrong credentials
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", "wrong", "wrong")
	zhtest.AssertEqual(t, http.StatusUnauthorized, rec.Code)

	// Test with correct credentials
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", "admin", "secret")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestNewWithDisabledEndpoints(t *testing.T) {
	cfg := DefaultConfig
	cfg.EnableIndex = config.Bool(false)
	cfg.EnableCmdline = config.Bool(false)
	app, _ := setupPProf(t, &cfg)

	// Test disabled index endpoint
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusNotFound, rec.Code)
}

func TestCmdlineEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/cmdline", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestSymbolEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	// Test GET
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/symbol", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Test POST
	rec = makeRequest(t, app, http.MethodPost, "/debug/pprof/symbol", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestHeapEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/heap", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestGoroutineEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/goroutine", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestThreadCreateEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/threadcreate", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestBlockEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/block", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestMutexEndpoint(t *testing.T) {
	app, _ := setupPProf(t, nil)

	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/mutex", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestAllEndpointsWithAuth(t *testing.T) {
	app, pp := setupPProfWithAuth(t, "test", "test")

	zhtest.AssertNotNil(t, pp.Auth)
	zhtest.AssertEqual(t, "test", pp.Auth.Username)
	zhtest.AssertEqual(t, "test", pp.Auth.Password)

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
		zhtest.AssertEqual(t, http.StatusUnauthorized, rec.Code)

		// With auth
		rec = makeRequest(t, app, http.MethodGet, endpoint, "test", "test")
		zhtest.AssertEqual(t, http.StatusOK, rec.Code)
	}
}

func TestAutoGeneratedAuth(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig // Auth is nil, so auto-generate
	pp := New(app, cfg)

	zhtest.AssertNotNil(t, pp.Auth)
	zhtest.AssertEqual(t, "pprof", pp.Auth.Username)
	zhtest.AssertNotEmpty(t, pp.Auth.Password)

	// Test that auth is required
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusUnauthorized, rec.Code)

	// Test with auto-generated credentials
	rec = makeRequest(t, app, http.MethodGet, "/debug/pprof/", pp.Auth.Username, pp.Auth.Password)
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestDisabledAuth(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // empty = disabled
	pp := New(app, cfg)

	zhtest.AssertNil(t, pp.Auth)

	// Test without auth - should succeed (from localhost)
	rec := makeRequest(t, app, http.MethodGet, "/debug/pprof/", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestIPAllowlistDefault(t *testing.T) {
	// Default config should only allow localhost
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // disable auth for this test
	New(app, cfg)

	// Request from localhost should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "127.0.0.1", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Request from external IP should be forbidden
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.1.100", "", "")
	zhtest.AssertEqual(t, http.StatusForbidden, rec.Code)
}

func TestIPAllowlistCustom(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // disable auth for this test
	cfg.AllowedIPs = []string{"192.168.1.0/24", "10.0.0.100"}
	New(app, cfg)

	// Request from allowed CIDR should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.1.50", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Request from allowed single IP should succeed
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "10.0.0.100", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Request from disallowed IP should be forbidden
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "192.168.2.1", "", "")
	zhtest.AssertEqual(t, http.StatusForbidden, rec.Code)
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
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
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
	zhtest.AssertEqual(t, http.StatusForbidden, rec.Code)

	// Request from allowed IP with wrong auth should get 401
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "127.0.0.1", "wrong", "wrong")
	zhtest.AssertEqual(t, http.StatusUnauthorized, rec.Code)

	// Request from allowed IP with correct auth should succeed
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "127.0.0.1", "admin", "secret")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)
}

func TestIPAllowlistIPv6(t *testing.T) {
	app := zh.New()
	cfg := DefaultConfig
	cfg.Auth = &AuthConfig{} // disable auth for this test
	cfg.AllowedIPs = []string{"::1/128", "2001:db8::/32"}
	New(app, cfg)

	// Request from localhost IPv6 should succeed
	rec := makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "::1", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Request from allowed IPv6 CIDR should succeed
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "2001:db8::1", "", "")
	zhtest.AssertEqual(t, http.StatusOK, rec.Code)

	// Request from disallowed IPv6 should be forbidden
	rec = makeRequestFromIP(t, app, http.MethodGet, "/debug/pprof/", "2001:db9::1", "", "")
	zhtest.AssertEqual(t, http.StatusForbidden, rec.Code)
}

// Test parseAllowedIPs with invalid IP
func TestParseAllowedIPs_InvalidIP(t *testing.T) {
	_, err := parseAllowedIPs([]string{"not-an-ip"})
	zhtest.AssertError(t, err)
}

// Test parseAllowedIPs with empty/whitespace IPs
func TestParseAllowedIPs_EmptyAndWhitespace(t *testing.T) {
	nets, err := parseAllowedIPs([]string{"", "  ", "127.0.0.1"})
	zhtest.AssertNoError(t, err)
	// Should skip empty entries and return only valid ones
	zhtest.AssertEqual(t, 1, len(nets))
}

// Test isIPAllowed with invalid IP
func TestIsIPAllowed_InvalidIP(t *testing.T) {
	nets, _ := parseAllowedIPs([]string{"127.0.0.1/32"})
	result := isIPAllowed("not-an-ip", nets)
	zhtest.AssertFalse(t, result)
}

// Test extractClientIP with trust proxy and X-Forwarded-For header
func TestExtractClientIP_WithXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "127.0.0.1:1234"

	ip := extractClientIP(req, true)
	zhtest.AssertEqual(t, "10.0.0.1", ip)
}

// Test extractClientIP with trust proxy and X-Real-IP header
func TestExtractClientIP_WithXRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "192.168.1.100")
	req.RemoteAddr = "127.0.0.1:1234"

	ip := extractClientIP(req, true)
	zhtest.AssertEqual(t, "192.168.1.100", ip)
}

// Test extractClientIP without trust proxy (falls back to RemoteAddr)
func TestExtractClientIP_WithoutTrustProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.RemoteAddr = "192.168.1.50:1234"

	ip := extractClientIP(req, false)
	zhtest.AssertEqual(t, "192.168.1.50", ip)
}

// Test extractClientIP with RemoteAddr without port
func TestExtractClientIP_RemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.50"

	ip := extractClientIP(req, false)
	zhtest.AssertEqual(t, "192.168.1.50", ip)
}

// Test extractClientIP X-Forwarded-For with single IP
func TestExtractClientIP_SingleXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.5")
	req.RemoteAddr = "127.0.0.1:1234"

	ip := extractClientIP(req, true)
	zhtest.AssertEqual(t, "10.0.0.5", ip)
}

// Test extractClientIP with empty X-Forwarded-For falls back to X-Real-IP
func TestExtractClientIP_EmptyXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "")
	req.Header.Set("X-Real-IP", "10.0.0.10")
	req.RemoteAddr = "127.0.0.1:1234"

	ip := extractClientIP(req, true)
	zhtest.AssertEqual(t, "10.0.0.10", ip)
}

// Test generateRandomPassword generates non-empty password
func TestGenerateRandomPassword(t *testing.T) {
	pw1 := generateRandomPassword()
	zhtest.AssertNotEmpty(t, pw1)

	// Should generate different passwords each time
	pw2 := generateRandomPassword()
	zhtest.AssertNotEqual(t, pw1, pw2)
}

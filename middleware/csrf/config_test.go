package csrf

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestCSRFConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig

	zhtest.AssertEqual(t, "csrf_token", cfg.CookieName)
	zhtest.AssertEqual(t, 86400, cfg.CookieMaxAge)
	zhtest.AssertEqual(t, "/", cfg.CookiePath)
	zhtest.AssertEmpty(t, cfg.CookieDomain)
	zhtest.AssertNotNil(t, cfg.CookieSecure)
	zhtest.AssertTrue(t, *cfg.CookieSecure)
	zhtest.AssertEqual(t, http.SameSiteStrictMode, cfg.CookieSameSite)
	zhtest.AssertEqual(t, "header:X-CSRF-Token", cfg.TokenLookup)
	zhtest.AssertNil(t, cfg.ErrorHandler)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	zhtest.AssertEqual(t, 4, len(cfg.ExcludedMethods))
	zhtest.AssertNil(t, cfg.HMACKey)
	zhtest.AssertNil(t, cfg.TokenGenerator)
}

func TestCSRFConfig_CustomValues(t *testing.T) {
	t.Run("custom cookie name", func(t *testing.T) {
		cfg := Config{
			CookieName: "my_csrf",
		}
		zhtest.AssertEqual(t, "my_csrf", cfg.CookieName)
	})

	t.Run("custom cookie max age", func(t *testing.T) {
		cfg := Config{
			CookieMaxAge: 3600,
		}
		zhtest.AssertEqual(t, 3600, cfg.CookieMaxAge)
	})

	t.Run("custom cookie path", func(t *testing.T) {
		cfg := Config{
			CookiePath: "/api",
		}
		zhtest.AssertEqual(t, "/api", cfg.CookiePath)
	})

	t.Run("custom cookie domain", func(t *testing.T) {
		cfg := Config{
			CookieDomain: "example.com",
		}
		zhtest.AssertEqual(t, "example.com", cfg.CookieDomain)
	})

	t.Run("custom cookie secure", func(t *testing.T) {
		cfg := Config{
			CookieSecure: config.Bool(false),
		}
		zhtest.AssertNotNil(t, cfg.CookieSecure)
		zhtest.AssertFalse(t, *cfg.CookieSecure)
	})

	t.Run("custom cookie same site", func(t *testing.T) {
		cfg := Config{
			CookieSameSite: http.SameSiteLaxMode,
		}
		zhtest.AssertEqual(t, http.SameSiteLaxMode, cfg.CookieSameSite)
	})

	t.Run("custom token lookup", func(t *testing.T) {
		cfg := Config{
			TokenLookup: "form:csrf_token",
		}
		zhtest.AssertEqual(t, "form:csrf_token", cfg.TokenLookup)
	})

	t.Run("custom HMAC key", func(t *testing.T) {
		key := []byte("test-key-that-is-32-bytes-long!!")
		cfg := Config{
			HMACKey: key,
		}
		zhtest.AssertEqual(t, string(key), string(cfg.HMACKey))
	})

	t.Run("custom token generator", func(t *testing.T) {
		generator := func(key []byte) (string, error) {
			return "test-token", nil
		}
		cfg := Config{
			TokenGenerator: generator,
		}
		zhtest.AssertNotNil(t, cfg.TokenGenerator)
	})
}

func TestCSRFConfig_IncludedPaths(t *testing.T) {
	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/api/public", "/health"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		zhtest.AssertEqual(t, "/api/public", cfg.IncludedPaths[0])
		zhtest.AssertEqual(t, "/health", cfg.IncludedPaths[1])
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.IncludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := Config{
			IncludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.IncludedPaths)
	})
}

func TestCSRFConfig_ExcludedMethods(t *testing.T) {
	cfg := DefaultConfig

	expectedMethods := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	for _, method := range cfg.ExcludedMethods {
		zhtest.AssertTrue(t, expectedMethods[method])
	}
}

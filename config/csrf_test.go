package config

import (
	"net/http"
	"testing"
)

func TestCSRFConfig_DefaultValues(t *testing.T) {
	cfg := DefaultCSRFConfig

	if cfg.CookieName != "csrf_token" {
		t.Errorf("expected default CookieName = csrf_token, got %s", cfg.CookieName)
	}

	if cfg.CookieMaxAge != 86400 {
		t.Errorf("expected default CookieMaxAge = 86400, got %d", cfg.CookieMaxAge)
	}

	if cfg.CookiePath != "/" {
		t.Errorf("expected default CookiePath = /, got %s", cfg.CookiePath)
	}

	if cfg.CookieDomain != "" {
		t.Errorf("expected default CookieDomain = empty, got %s", cfg.CookieDomain)
	}

	if cfg.CookieSecure == nil || !*cfg.CookieSecure {
		t.Error("expected default CookieSecure = true")
	}

	if cfg.CookieSameSite != http.SameSiteStrictMode {
		t.Errorf("expected default CookieSameSite = Strict, got %v", cfg.CookieSameSite)
	}

	if cfg.TokenLookup != "header:X-CSRF-Token" {
		t.Errorf("expected default TokenLookup = header:X-CSRF-Token, got %s", cfg.TokenLookup)
	}

	if cfg.ErrorHandler != nil {
		t.Error("expected default ErrorHandler to be nil")
	}

	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default ExemptPaths to be empty, got %d paths", len(cfg.ExemptPaths))
	}

	if len(cfg.ExemptMethods) != 4 {
		t.Errorf("expected default ExemptMethods to have 4 methods, got %d", len(cfg.ExemptMethods))
	}

	if cfg.HMACKey != nil {
		t.Error("expected default HMACKey to be nil")
	}

	if cfg.TokenGenerator != nil {
		t.Error("expected default TokenGenerator to be nil")
	}
}

func TestCSRFConfig_CustomValues(t *testing.T) {
	t.Run("custom cookie name", func(t *testing.T) {
		cfg := CSRFConfig{
			CookieName: "my_csrf",
		}
		if cfg.CookieName != "my_csrf" {
			t.Errorf("expected CookieName = my_csrf, got %s", cfg.CookieName)
		}
	})

	t.Run("custom cookie max age", func(t *testing.T) {
		cfg := CSRFConfig{
			CookieMaxAge: 3600,
		}
		if cfg.CookieMaxAge != 3600 {
			t.Errorf("expected CookieMaxAge = 3600, got %d", cfg.CookieMaxAge)
		}
	})

	t.Run("custom cookie path", func(t *testing.T) {
		cfg := CSRFConfig{
			CookiePath: "/api",
		}
		if cfg.CookiePath != "/api" {
			t.Errorf("expected CookiePath = /api, got %s", cfg.CookiePath)
		}
	})

	t.Run("custom cookie domain", func(t *testing.T) {
		cfg := CSRFConfig{
			CookieDomain: "example.com",
		}
		if cfg.CookieDomain != "example.com" {
			t.Errorf("expected CookieDomain = example.com, got %s", cfg.CookieDomain)
		}
	})

	t.Run("custom cookie secure", func(t *testing.T) {
		cfg := CSRFConfig{
			CookieSecure: Bool(false),
		}
		if cfg.CookieSecure == nil || *cfg.CookieSecure != false {
			t.Error("expected CookieSecure = false")
		}
	})

	t.Run("custom cookie same site", func(t *testing.T) {
		cfg := CSRFConfig{
			CookieSameSite: http.SameSiteLaxMode,
		}
		if cfg.CookieSameSite != http.SameSiteLaxMode {
			t.Errorf("expected CookieSameSite = Lax, got %v", cfg.CookieSameSite)
		}
	})

	t.Run("custom token lookup", func(t *testing.T) {
		cfg := CSRFConfig{
			TokenLookup: "form:csrf_token",
		}
		if cfg.TokenLookup != "form:csrf_token" {
			t.Errorf("expected TokenLookup = form:csrf_token, got %s", cfg.TokenLookup)
		}
	})

	t.Run("custom HMAC key", func(t *testing.T) {
		key := []byte("test-key-that-is-32-bytes-long!!")
		cfg := CSRFConfig{
			HMACKey: key,
		}
		if string(cfg.HMACKey) != string(key) {
			t.Error("expected HMACKey to be set correctly")
		}
	})

	t.Run("custom token generator", func(t *testing.T) {
		generator := func(key []byte) (string, error) {
			return "test-token", nil
		}
		cfg := CSRFConfig{
			TokenGenerator: generator,
		}
		if cfg.TokenGenerator == nil {
			t.Error("expected TokenGenerator to be set")
		}
	})
}

func TestCSRFConfig_ExemptMethods(t *testing.T) {
	cfg := DefaultCSRFConfig

	expectedMethods := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}

	for _, method := range cfg.ExemptMethods {
		if !expectedMethods[method] {
			t.Errorf("unexpected exempt method: %s", method)
		}
	}
}

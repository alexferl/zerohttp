package config

import (
	"testing"
)

func TestBasicAuthConfig_DefaultValues(t *testing.T) {
	cfg := DefaultBasicAuthConfig
	if cfg.Realm != "Restricted" {
		t.Errorf("expected default realm = Restricted, got %s", cfg.Realm)
	}
	if cfg.Credentials != nil {
		t.Error("expected default credentials to be nil")
	}
	if cfg.Validator != nil {
		t.Error("expected default validator to be nil")
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("expected default exempt paths to be empty, got %d paths", len(cfg.ExemptPaths))
	}
}

func TestBasicAuthOptions(t *testing.T) {
	t.Run("realm option", func(t *testing.T) {
		cfg := DefaultBasicAuthConfig
		WithBasicAuthRealm("Admin Area")(&cfg)
		if cfg.Realm != "Admin Area" {
			t.Errorf("expected realm = Admin Area, got %s", cfg.Realm)
		}
	})

	t.Run("credentials option", func(t *testing.T) {
		credentials := map[string]string{
			"admin":    "password123",
			"user":     "userpass",
			"readonly": "readonlypass",
		}
		cfg := DefaultBasicAuthConfig
		WithBasicAuthCredentials(credentials)(&cfg)
		if cfg.Credentials == nil {
			t.Error("expected credentials to be set")
		}
		if len(cfg.Credentials) != 3 {
			t.Errorf("expected 3 credentials, got %d", len(cfg.Credentials))
		}
		for user, pass := range credentials {
			if cfg.Credentials[user] != pass {
				t.Errorf("expected %s password = %s, got %s", user, pass, cfg.Credentials[user])
			}
		}
	})

	t.Run("validator option", func(t *testing.T) {
		mockValidator := func(username, password string) bool {
			return username == "testuser" && password == "testpass"
		}
		cfg := DefaultBasicAuthConfig
		WithBasicAuthValidator(mockValidator)(&cfg)
		if cfg.Validator == nil {
			t.Error("expected validator to be set")
		}
		if !cfg.Validator("testuser", "testpass") {
			t.Error("expected validator to return true for valid credentials")
		}
		if cfg.Validator("wronguser", "wrongpass") {
			t.Error("expected validator to return false for invalid credentials")
		}
	})

	t.Run("exempt paths option", func(t *testing.T) {
		exemptPaths := []string{"/health", "/metrics", "/login", "/signup"}
		cfg := DefaultBasicAuthConfig
		WithBasicAuthExemptPaths(exemptPaths)(&cfg)
		if len(cfg.ExemptPaths) != 4 {
			t.Errorf("expected 4 exempt paths, got %d", len(cfg.ExemptPaths))
		}
		expectedPaths := map[string]bool{
			"/health": true, "/metrics": true, "/login": true, "/signup": true,
		}
		for _, path := range cfg.ExemptPaths {
			if !expectedPaths[path] {
				t.Errorf("unexpected exempt path: %s", path)
			}
		}
	})
}

func TestBasicAuthConfig_MultipleOptions(t *testing.T) {
	credentials := map[string]string{"admin": "secret123", "user": "pass456"}
	exemptPaths := []string{"/public", "/health"}
	validator := func(username, password string) bool {
		return username == "custom" && password == "validate"
	}

	cfg := DefaultBasicAuthConfig
	WithBasicAuthRealm("Custom Realm")(&cfg)
	WithBasicAuthCredentials(credentials)(&cfg)
	WithBasicAuthValidator(validator)(&cfg)
	WithBasicAuthExemptPaths(exemptPaths)(&cfg)

	if cfg.Realm != "Custom Realm" {
		t.Errorf("expected realm = Custom Realm, got %s", cfg.Realm)
	}
	if len(cfg.Credentials) != 2 {
		t.Errorf("expected 2 credentials, got %d", len(cfg.Credentials))
	}
	if cfg.Credentials["admin"] != "secret123" {
		t.Error("expected admin credentials to be set correctly")
	}
	if cfg.Validator == nil {
		t.Error("expected validator to be set")
	}
	if !cfg.Validator("custom", "validate") {
		t.Error("expected custom validator to work")
	}
	if len(cfg.ExemptPaths) != 2 {
		t.Errorf("expected 2 exempt paths, got %d", len(cfg.ExemptPaths))
	}
}

func TestBasicAuthConfig_EdgeCases(t *testing.T) {
	t.Run("empty credentials", func(t *testing.T) {
		cfg := DefaultBasicAuthConfig
		WithBasicAuthCredentials(map[string]string{})(&cfg)
		if cfg.Credentials == nil {
			t.Error("expected credentials map to be initialized, not nil")
		}
		if len(cfg.Credentials) != 0 {
			t.Errorf("expected empty credentials map, got %d entries", len(cfg.Credentials))
		}
	})

	t.Run("nil credentials", func(t *testing.T) {
		cfg := DefaultBasicAuthConfig
		WithBasicAuthCredentials(nil)(&cfg)
		if cfg.Credentials != nil {
			t.Error("expected credentials to remain nil when nil is passed")
		}
	})

	t.Run("empty exempt paths", func(t *testing.T) {
		cfg := DefaultBasicAuthConfig
		WithBasicAuthExemptPaths([]string{})(&cfg)
		if cfg.ExemptPaths == nil {
			t.Error("expected exempt paths slice to be initialized, not nil")
		}
		if len(cfg.ExemptPaths) != 0 {
			t.Errorf("expected empty exempt paths slice, got %d entries", len(cfg.ExemptPaths))
		}
	})

	t.Run("nil exempt paths", func(t *testing.T) {
		cfg := DefaultBasicAuthConfig
		WithBasicAuthExemptPaths(nil)(&cfg)
		if cfg.ExemptPaths != nil {
			t.Error("expected exempt paths to remain nil when nil is passed")
		}
	})
}

func TestBasicAuthConfig_ValidatorFunctionality(t *testing.T) {
	users := map[string]string{
		"alice":   "alice123",
		"bob":     "bob456",
		"charlie": "charlie789",
	}

	validator := func(username, password string) bool {
		expectedPassword, exists := users[username]
		return exists && expectedPassword == password
	}

	cfg := DefaultBasicAuthConfig
	WithBasicAuthValidator(validator)(&cfg)

	testCases := []struct {
		username, password string
		expected           bool
	}{
		{"alice", "alice123", true},
		{"bob", "bob456", true},
		{"charlie", "charlie789", true},
		{"alice", "wrongpass", false},
		{"nonexistent", "anypass", false},
		{"", "", false},
	}

	for _, tc := range testCases {
		result := cfg.Validator(tc.username, tc.password)
		if result != tc.expected {
			t.Errorf("validator(%q, %q) = %v, expected %v",
				tc.username, tc.password, result, tc.expected)
		}
	}
}

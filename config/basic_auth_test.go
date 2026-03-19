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
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("expected default excluded paths to be empty, got %d paths", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 0 {
		t.Errorf("expected default included paths to be empty, got %d paths", len(cfg.IncludedPaths))
	}
}

func TestBasicAuthConfig_CustomValues(t *testing.T) {
	t.Run("custom realm", func(t *testing.T) {
		cfg := BasicAuthConfig{
			Realm: "Admin Area",
		}
		if cfg.Realm != "Admin Area" {
			t.Errorf("expected realm = Admin Area, got %s", cfg.Realm)
		}
	})

	t.Run("custom credentials", func(t *testing.T) {
		credentials := map[string]string{
			"admin":    "password123",
			"user":     "userpass",
			"readonly": "readonlypass",
		}
		cfg := BasicAuthConfig{
			Credentials: credentials,
		}
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

	t.Run("custom validator", func(t *testing.T) {
		mockValidator := func(username, password string) bool {
			return username == "testuser" && password == "testpass"
		}
		cfg := BasicAuthConfig{
			Validator: mockValidator,
		}
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

	t.Run("custom excluded paths", func(t *testing.T) {
		excludedPaths := []string{"/health", "/metrics", "/login", "/signup"}
		cfg := BasicAuthConfig{
			ExcludedPaths: excludedPaths,
		}
		if len(cfg.ExcludedPaths) != 4 {
			t.Errorf("expected 4 excluded paths, got %d", len(cfg.ExcludedPaths))
		}
		expectedPaths := map[string]bool{
			"/health": true, "/metrics": true, "/login": true, "/signup": true,
		}
		for _, path := range cfg.ExcludedPaths {
			if !expectedPaths[path] {
				t.Errorf("unexpected excluded path: %s", path)
			}
		}
	})

	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/admin", "/api/private/"}
		cfg := BasicAuthConfig{
			IncludedPaths: includedPaths,
		}
		if len(cfg.IncludedPaths) != 2 {
			t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
		}
		expectedPaths := map[string]bool{
			"/admin": true, "/api/private/": true,
		}
		for _, path := range cfg.IncludedPaths {
			if !expectedPaths[path] {
				t.Errorf("unexpected allowed path: %s", path)
			}
		}
	})
}

func TestBasicAuthConfig_MultipleFields(t *testing.T) {
	credentials := map[string]string{"admin": "secret123", "user": "pass456"}
	excludedPaths := []string{"/public", "/health"}
	includedPaths := []string{"/admin", "/api/"}
	validator := func(username, password string) bool {
		return username == "custom" && password == "validate"
	}

	cfg := BasicAuthConfig{
		Realm:         "Custom Realm",
		Credentials:   credentials,
		Validator:     validator,
		ExcludedPaths: excludedPaths,
		IncludedPaths: includedPaths,
	}

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
	if len(cfg.ExcludedPaths) != 2 {
		t.Errorf("expected 2 excluded paths, got %d", len(cfg.ExcludedPaths))
	}
	if len(cfg.IncludedPaths) != 2 {
		t.Errorf("expected 2 included paths, got %d", len(cfg.IncludedPaths))
	}
}

func TestBasicAuthConfig_EdgeCases(t *testing.T) {
	t.Run("empty credentials", func(t *testing.T) {
		cfg := BasicAuthConfig{
			Credentials: map[string]string{},
		}
		if cfg.Credentials == nil {
			t.Error("expected credentials map to be initialized, not nil")
		}
		if len(cfg.Credentials) != 0 {
			t.Errorf("expected empty credentials map, got %d entries", len(cfg.Credentials))
		}
	})

	t.Run("nil credentials", func(t *testing.T) {
		cfg := BasicAuthConfig{
			Credentials: nil,
		}
		if cfg.Credentials != nil {
			t.Error("expected credentials to remain nil when nil is passed")
		}
	})

	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := BasicAuthConfig{
			ExcludedPaths: []string{},
		}
		if cfg.ExcludedPaths == nil {
			t.Error("expected excluded paths slice to be initialized, not nil")
		}
		if len(cfg.ExcludedPaths) != 0 {
			t.Errorf("expected empty excluded paths slice, got %d entries", len(cfg.ExcludedPaths))
		}
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := BasicAuthConfig{
			ExcludedPaths: nil,
		}
		if cfg.ExcludedPaths != nil {
			t.Error("expected excluded paths to remain nil when nil is passed")
		}
	})

	t.Run("empty included paths", func(t *testing.T) {
		cfg := BasicAuthConfig{
			IncludedPaths: []string{},
		}
		if cfg.IncludedPaths == nil {
			t.Error("expected included paths slice to be initialized, not nil")
		}
		if len(cfg.IncludedPaths) != 0 {
			t.Errorf("expected empty included paths slice, got %d entries", len(cfg.IncludedPaths))
		}
	})

	t.Run("nil included paths", func(t *testing.T) {
		cfg := BasicAuthConfig{
			IncludedPaths: nil,
		}
		if cfg.IncludedPaths != nil {
			t.Error("expected included paths to remain nil when nil is passed")
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

	cfg := BasicAuthConfig{
		Validator: validator,
	}

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

package basicauth

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestBasicAuthConfig_DefaultValues(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, "Restricted", cfg.Realm)
	zhtest.AssertNil(t, cfg.Credentials)
	zhtest.AssertNil(t, cfg.Validator)
	zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 0, len(cfg.IncludedPaths))
}

func TestBasicAuthConfig_CustomValues(t *testing.T) {
	t.Run("custom realm", func(t *testing.T) {
		cfg := Config{
			Realm: "Admin Area",
		}
		zhtest.AssertEqual(t, "Admin Area", cfg.Realm)
	})

	t.Run("custom credentials", func(t *testing.T) {
		credentials := map[string]string{
			"admin":    "password123",
			"user":     "userpass",
			"readonly": "readonlypass",
		}
		cfg := Config{
			Credentials: credentials,
		}
		zhtest.AssertNotNil(t, cfg.Credentials)
		zhtest.AssertEqual(t, 3, len(cfg.Credentials))
		for user, pass := range credentials {
			zhtest.AssertEqual(t, pass, cfg.Credentials[user])
		}
	})

	t.Run("custom validator", func(t *testing.T) {
		mockValidator := func(username, password string) bool {
			return username == "testuser" && password == "testpass"
		}
		cfg := Config{
			Validator: mockValidator,
		}
		zhtest.AssertNotNil(t, cfg.Validator)
		zhtest.AssertTrue(t, cfg.Validator("testuser", "testpass"))
		zhtest.AssertFalse(t, cfg.Validator("wronguser", "wrongpass"))
	})

	t.Run("custom excluded paths", func(t *testing.T) {
		excludedPaths := []string{"/health", "/metrics", "/login", "/signup"}
		cfg := Config{
			ExcludedPaths: excludedPaths,
		}
		zhtest.AssertEqual(t, 4, len(cfg.ExcludedPaths))
		expectedPaths := map[string]bool{
			"/health": true, "/metrics": true, "/login": true, "/signup": true,
		}
		for _, path := range cfg.ExcludedPaths {
			zhtest.AssertTrue(t, expectedPaths[path])
		}
	})

	t.Run("custom included paths", func(t *testing.T) {
		includedPaths := []string{"/admin", "/api/private/"}
		cfg := Config{
			IncludedPaths: includedPaths,
		}
		zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
		expectedPaths := map[string]bool{
			"/admin": true, "/api/private/": true,
		}
		for _, path := range cfg.IncludedPaths {
			zhtest.AssertTrue(t, expectedPaths[path])
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

	cfg := Config{
		Realm:         "Custom Realm",
		Credentials:   credentials,
		Validator:     validator,
		ExcludedPaths: excludedPaths,
		IncludedPaths: includedPaths,
	}

	zhtest.AssertEqual(t, "Custom Realm", cfg.Realm)
	zhtest.AssertEqual(t, 2, len(cfg.Credentials))
	zhtest.AssertEqual(t, "secret123", cfg.Credentials["admin"])
	zhtest.AssertNotNil(t, cfg.Validator)
	zhtest.AssertTrue(t, cfg.Validator("custom", "validate"))
	zhtest.AssertEqual(t, 2, len(cfg.ExcludedPaths))
	zhtest.AssertEqual(t, 2, len(cfg.IncludedPaths))
}

func TestBasicAuthConfig_EdgeCases(t *testing.T) {
	t.Run("empty credentials", func(t *testing.T) {
		cfg := Config{
			Credentials: map[string]string{},
		}
		zhtest.AssertNotNil(t, cfg.Credentials)
		zhtest.AssertEqual(t, 0, len(cfg.Credentials))
	})

	t.Run("nil credentials", func(t *testing.T) {
		cfg := Config{
			Credentials: nil,
		}
		zhtest.AssertNil(t, cfg.Credentials)
	})

	t.Run("empty excluded paths", func(t *testing.T) {
		cfg := Config{
			ExcludedPaths: []string{},
		}
		zhtest.AssertNotNil(t, cfg.ExcludedPaths)
		zhtest.AssertEqual(t, 0, len(cfg.ExcludedPaths))
	})

	t.Run("nil excluded paths", func(t *testing.T) {
		cfg := Config{
			ExcludedPaths: nil,
		}
		zhtest.AssertNil(t, cfg.ExcludedPaths)
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

	cfg := Config{
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
		zhtest.AssertEqual(t, tc.expected, result)
	}
}

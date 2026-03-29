package mwutil

import (
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestPathMatches(t *testing.T) {
	tests := []struct {
		requestPath  string
		excludedPath string
		expected     bool
	}{
		{"/health", "/health", true},
		{"/health", "/metrics", false},
		{"/api/public/users", "/api/public/", true},
		{"/api/public", "/api/public/", true},   // path without trailing slash matches
		{"/api/public/", "/api/public/", true},  // exact match with trailing slash
		{"/api/publicx", "/api/public/", false}, // different path, shouldn't match
		{"/", "/", true},
		{"", "", true},
		{"/api/v1/users", "/api/", true},
		{"/api/v1/users", "/api", false}, // no trailing slash = no prefix match
		{"/api", "/api/", true},          // path without trailing slash matches
		{"/different", "/api/", false},
		// Wildcard suffix tests
		{"/api/live", "/api/live*", true},
		{"/api/livez", "/api/live*", true},
		{"/api/health/live", "/api/live*", false}, // doesn't start with /api/live
		{"/api", "/api*", true},
		{"/api/v1", "/api*", true},
	}

	for _, tt := range tests {
		t.Run(tt.requestPath+"_vs_"+tt.excludedPath, func(t *testing.T) {
			result := PathMatches(tt.requestPath, tt.excludedPath)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestShouldProcessMiddleware(t *testing.T) {
	t.Run("no paths set - process all", func(t *testing.T) {
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/any/path", nil, nil))
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/health", []string{}, []string{}))
	})

	t.Run("with included paths - only process matches", func(t *testing.T) {
		included := []string{"/api/", "/public"}
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/api/users", included, nil))
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/public", included, nil))
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/api/v1/items", included, nil))
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/health", included, nil))
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/metrics", included, nil))
	})

	t.Run("with excluded paths - process non-matches", func(t *testing.T) {
		excluded := []string{"/health", "/metrics"}
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/api/users", nil, excluded))
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/public", nil, excluded))
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/health", nil, excluded))
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/metrics", nil, excluded))
	})

	t.Run("included paths with wildcards", func(t *testing.T) {
		included := []string{"/api/*"}
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/api/v1", included, nil))
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/api/v2/users", included, nil))
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/health", included, nil))
	})

	t.Run("excluded paths with wildcards", func(t *testing.T) {
		excluded := []string{"/api/*"}
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/api/v1", nil, excluded))
		zhtest.AssertFalse(t, ShouldProcessMiddleware("/api/v2/users", nil, excluded))
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/health", nil, excluded))
	})

	t.Run("included paths take precedence", func(t *testing.T) {
		// When included paths is set, excluded paths are ignored
		included := []string{"/api/"}
		excluded := []string{"/api/internal"}
		// Should process /api/internal because it's in included paths
		zhtest.AssertTrue(t, ShouldProcessMiddleware("/api/internal", included, excluded))
	})
}

func TestValidatePathConfig(t *testing.T) {
	t.Run("valid config - only excluded paths", func(t *testing.T) {
		// Should not panic
		ValidatePathConfig([]string{"/health"}, nil, "TestMiddleware")
	})

	t.Run("valid config - only included paths", func(t *testing.T) {
		// Should not panic
		ValidatePathConfig(nil, []string{"/api/"}, "TestMiddleware")
	})

	t.Run("valid config - neither set", func(t *testing.T) {
		// Should not panic
		ValidatePathConfig(nil, nil, "TestMiddleware")
		ValidatePathConfig([]string{}, []string{}, "TestMiddleware")
	})

	t.Run("invalid config - both set", func(t *testing.T) {
		// Should panic
		zhtest.AssertPanic(t, func() {
			ValidatePathConfig([]string{"/health"}, []string{"/api/"}, "TestMiddleware")
		})
	})

	t.Run("panic message contains middleware name", func(t *testing.T) {
		zhtest.AssertPanicContains(t, func() {
			ValidatePathConfig([]string{"/health"}, []string{"/api/"}, "MyCustomMiddleware")
		}, "MyCustomMiddleware")
	})
}

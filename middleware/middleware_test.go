package middleware

import (
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestDefaultMiddlewares(t *testing.T) {
	cfg := config.DefaultConfig
	logger := &mockLogger{}

	middlewares := DefaultMiddlewares(cfg, logger)

	expectedCount := 5
	if len(middlewares) != expectedCount {
		t.Errorf("Expected %d middlewares, got %d", expectedCount, len(middlewares))
	}

	for i, middleware := range middlewares {
		if middleware == nil {
			t.Errorf("Middleware at index %d is nil", i)
		}
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var wrappedHandler http.Handler = baseHandler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrappedHandler = middlewares[i](wrappedHandler)
	}

	if wrappedHandler == nil {
		t.Error("Wrapped handler should not be nil")
	}

	req := zhtest.NewRequest(http.MethodGet, "/").Build()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Middleware chain panicked: %v", r)
		}
	}()

	w := zhtest.Serve(wrappedHandler, req)

	if w.Code == 0 {
		t.Error("Expected response code to be set")
	}
}

func TestDefaultMiddlewares_NilInputs(t *testing.T) {
	t.Run("nil logger", func(t *testing.T) {
		cfg := config.Config{}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("DefaultMiddlewares panicked with nil logger: %v", r)
			}
		}()

		middlewares := DefaultMiddlewares(cfg, nil)

		if len(middlewares) == 0 {
			t.Error("Expected middlewares to be returned even with nil logger")
		}
	})
}

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
	}

	for _, tt := range tests {
		t.Run(tt.requestPath+"_vs_"+tt.excludedPath, func(t *testing.T) {
			result := pathMatches(tt.requestPath, tt.excludedPath)
			if result != tt.expected {
				t.Errorf("pathMatches(%q, %q) = %v, expected %v",
					tt.requestPath, tt.excludedPath, result, tt.expected)
			}
		})
	}
}

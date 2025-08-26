package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestDefaultMiddlewares(t *testing.T) {
	cfg := config.Config{
		RequestIDOptions:       []config.RequestIDOption{},
		RecoverOptions:         []config.RecoverOption{},
		RequestBodySizeOptions: []config.RequestBodySizeOption{},
		SecurityHeadersOptions: []config.SecurityHeadersOption{},
		RequestLoggerOptions:   []config.RequestLoggerOption{},
	}
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

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Middleware chain panicked: %v", r)
		}
	}()

	wrappedHandler.ServeHTTP(w, req)

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
		requestPath string
		exemptPath  string
		expected    bool
	}{
		{"/health", "/health", true},
		{"/health", "/metrics", false},
		{"/api/public/users", "/api/public/", true},
		{"/api/public", "/api/public/", false},
		{"/api/publicx", "/api/public/", false},
		{"/", "/", true},
		{"", "", true},
		{"/api/v1/users", "/api/", true},
		{"/api/v1/users", "/api", false},
		{"/different", "/api/", false},
	}

	for _, tt := range tests {
		t.Run(tt.requestPath+"_vs_"+tt.exemptPath, func(t *testing.T) {
			result := pathMatches(tt.requestPath, tt.exemptPath)
			if result != tt.expected {
				t.Errorf("pathMatches(%q, %q) = %v, expected %v",
					tt.requestPath, tt.exemptPath, result, tt.expected)
			}
		})
	}
}

package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

type requestBodySizeTestHandler struct {
	called    bool
	bodyRead  []byte
	bodyError error
}

func (h *requestBodySizeTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	body, err := io.ReadAll(r.Body)
	h.bodyRead = body
	h.bodyError = err
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		panic(fmt.Errorf("failed to write test response: %w", err))
	}
}

func TestRequestBodySize_Limits(t *testing.T) {
	tests := []struct {
		name        string
		maxBytes    int64
		bodyContent string
		expectError bool
	}{
		{"exceeds limit", 10, "this body is definitely longer than 10 bytes", true},
		{"exact limit", 10, "1234567890", false},
		{"under limit", 20, "short", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := RequestBodySize(config.WithRequestBodySizeMaxBytes(tt.maxBytes))(handler)
			body := bytes.NewReader([]byte(tt.bodyContent))
			req := httptest.NewRequest("POST", "/", body)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if !handler.called {
				t.Error("Expected handler to be called")
			}
			hasError := handler.bodyError != nil
			if tt.expectError && !hasError {
				t.Error("Expected error when body exceeds limit")
			}
			if !tt.expectError && hasError {
				t.Errorf("Expected no error, got %v", handler.bodyError)
			}
			if tt.expectError && hasError && !strings.Contains(handler.bodyError.Error(), "too large") {
				t.Errorf("Expected 'too large' error, got: %v", handler.bodyError)
			}
		})
	}
}

func TestRequestBodySize_ExemptPaths(t *testing.T) {
	tests := []struct {
		path        string
		expectError bool
	}{
		{"/upload", false},
		{"/webhook", false},
		{"/api/large", false},
		{"/api/small", true},
		{"/other", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := RequestBodySize(
				config.WithRequestBodySizeMaxBytes(5),
				config.WithRequestBodySizeExemptPaths([]string{"/upload", "/webhook", "/api/large"}),
			)(handler)
			largeBody := bytes.NewReader([]byte("this is a long body"))
			req := httptest.NewRequest("POST", tt.path, largeBody)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if !handler.called {
				t.Errorf("Expected handler to be called for path %s", tt.path)
			}
			hasError := handler.bodyError != nil
			if !tt.expectError && hasError {
				t.Errorf("Expected no error for exempt path %s, got %v", tt.path, handler.bodyError)
			}
			if tt.expectError && !hasError {
				t.Errorf("Expected error for non-exempt path %s, got none", tt.path)
			}
		})
	}
}

func TestRequestBodySize_EmptyExemptPaths(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(
		config.WithRequestBodySizeMaxBytes(10),
		config.WithRequestBodySizeExemptPaths([]string{}),
	)(handler)
	largeBody := bytes.NewReader([]byte("this body is longer than 10 bytes"))
	req := httptest.NewRequest("POST", "/any-path", largeBody)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError == nil {
		t.Error("Expected error when no paths are exempt")
	}
}

func TestRequestBodySize_ConfigFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		maxBytes int64
	}{
		{"invalid negative", -1},
		{"zero max bytes", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &requestBodySizeTestHandler{}
			middleware := RequestBodySize(config.WithRequestBodySizeMaxBytes(tt.maxBytes))(handler)
			smallBody := bytes.NewReader([]byte("small body"))
			req := httptest.NewRequest("POST", "/", smallBody)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if !handler.called {
				t.Error("Expected handler to be called")
			}
			if handler.bodyError != nil {
				t.Errorf("Expected no error with default config, got %v", handler.bodyError)
			}
		})
	}
}

func TestRequestBodySize_NilExemptPaths(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(
		config.WithRequestBodySizeMaxBytes(100),
		config.WithRequestBodySizeExemptPaths(nil),
	)(handler)
	smallBody := bytes.NewReader([]byte("small body"))
	req := httptest.NewRequest("POST", "/", smallBody)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError != nil {
		t.Errorf("Expected no error with nil exempt paths, got %v", handler.bodyError)
	}
}

func TestRequestBodySize_MultipleOptions(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(
		config.WithRequestBodySizeMaxBytes(10),
		config.WithRequestBodySizeExemptPaths([]string{"/test"}),
	)(handler)
	largeBody := bytes.NewReader([]byte("this body is longer than 10 bytes but less than 100"))
	req := httptest.NewRequest("POST", "/", largeBody)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError == nil {
		t.Error("Expected error with 10 byte limit")
	}
}

func TestDefaultRequestBodySizeConfig(t *testing.T) {
	cfg := config.DefaultRequestBodySizeConfig
	expectedMaxBytes := int64(1 << 20)
	if cfg.MaxBytes != expectedMaxBytes {
		t.Errorf("Expected default max bytes %d, got %d", expectedMaxBytes, cfg.MaxBytes)
	}
	if cfg.ExemptPaths == nil {
		t.Error("Expected default exempt paths to be empty slice, got nil")
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("Expected default exempt paths to be empty, got %d items", len(cfg.ExemptPaths))
	}
}

func TestRequestBodySize_GetRequest(t *testing.T) {
	handler := &requestBodySizeTestHandler{}
	middleware := RequestBodySize(config.WithRequestBodySizeMaxBytes(10))(handler)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if !handler.called {
		t.Error("Expected handler to be called")
	}
	if handler.bodyError != nil {
		t.Errorf("Expected no error for GET request, got %v", handler.bodyError)
	}
}

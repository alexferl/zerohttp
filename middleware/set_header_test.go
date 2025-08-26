package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
)

func TestSetHeader_DefaultConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	middleware := SetHeader()(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if len(w.Header()) > 1 { // Content-Length is always present
		t.Errorf("Expected no custom headers with default config, got %d headers", len(w.Header()))
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestSetHeader_SingleHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{
		"X-Custom-Header": "custom-value",
	}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if headerValue := w.Header().Get("X-Custom-Header"); headerValue != "custom-value" {
		t.Errorf("Expected header 'X-Custom-Header' to be 'custom-value', got '%s'", headerValue)
	}
}

func TestSetHeader_MultipleHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	headers := map[string]string{
		"X-Custom-Header":  "custom-value",
		"X-Another-Header": "another-value",
		"Cache-Control":    "no-cache",
		"X-API-Version":    "v1.0",
	}
	middleware := SetHeader(config.WithSetHeaders(headers))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	for expectedKey, expectedValue := range headers {
		actualValue := w.Header().Get(expectedKey)
		if actualValue != expectedValue {
			t.Errorf("Expected header '%s' to be '%s', got '%s'", expectedKey, expectedValue, actualValue)
		}
	}
}

func TestSetHeader_EmptyHeaderValue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{
		"X-Empty-Header":  "",
		"X-Normal-Header": "normal-value",
	}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	emptyHeaderValue := w.Header().Get("X-Empty-Header")
	if emptyHeaderValue != "" {
		t.Errorf("Expected empty header 'X-Empty-Header' to be '', got '%s'", emptyHeaderValue)
	}
	_, exists := w.Header()["X-Empty-Header"]
	if !exists {
		t.Error("Expected empty header 'X-Empty-Header' to exist")
	}
	if normalHeaderValue := w.Header().Get("X-Normal-Header"); normalHeaderValue != "normal-value" {
		t.Errorf("Expected header 'X-Normal-Header' to be 'normal-value', got '%s'", normalHeaderValue)
	}
}

func TestSetHeader_NilHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(nil))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSetHeader_OverrideExistingHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Server", "Default-Server")
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{
		"Content-Type": "application/json",
		"Server":       "Custom-Server",
	}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if contentType := w.Header().Get("Content-Type"); contentType != "text/html" {
		t.Errorf("Expected Content-Type to be overridden to 'text/html', got '%s'", contentType)
	}
	if server := w.Header().Get("Server"); server != "Default-Server" {
		t.Errorf("Expected Server to be overridden to 'Default-Server', got '%s'", server)
	}
}

func TestSetHeader_HeadersSetBeforeHandler(t *testing.T) {
	var headerValueInHandler string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerValueInHandler = w.Header().Get("X-Middleware-Header")
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{
		"X-Middleware-Header": "middleware-value",
	}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if headerValueInHandler != "middleware-value" {
		t.Errorf("Expected header to be visible in handler as 'middleware-value', got '%s'", headerValueInHandler)
	}
	finalHeaderValue := w.Header().Get("X-Middleware-Header")
	if finalHeaderValue != "middleware-value" {
		t.Errorf("Expected final header to be 'middleware-value', got '%s'", finalHeaderValue)
	}
}

func TestSetHeader_CaseInsensitiveHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{
		"content-type":    "application/json",
		"x-custom-header": "lowercase-key",
	}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type to be 'application/json', got '%s'", contentType)
	}
	if customHeader := w.Header().Get("X-Custom-Header"); customHeader != "lowercase-key" {
		t.Errorf("Expected X-Custom-Header to be 'lowercase-key', got '%s'", customHeader)
	}
}

func TestSetHeader_SpecialCharactersInHeaderValue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{
		"X-Special-Chars": "value with spaces, commas; and: colons",
		"X-Unicode":       "测试值",
		"X-Numbers":       "12345",
	}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if specialChars := w.Header().Get("X-Special-Chars"); specialChars != "value with spaces, commas; and: colons" {
		t.Errorf("Expected special characters header to be preserved, got '%s'", specialChars)
	}
	if unicode := w.Header().Get("X-Unicode"); unicode != "测试值" {
		t.Errorf("Expected unicode header to be preserved, got '%s'", unicode)
	}
	if numbers := w.Header().Get("X-Numbers"); numbers != "12345" {
		t.Errorf("Expected numbers header to be '12345', got '%s'", numbers)
	}
}

func TestSetHeader_MultipleOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(
		config.WithSetHeaders(map[string]string{"X-First-Header": "first-value"}),
		config.WithSetHeaders(map[string]string{"X-Second-Header": "second-value"}),
	)(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if secondHeader := w.Header().Get("X-Second-Header"); secondHeader != "second-value" {
		t.Errorf("Expected last option header 'X-Second-Header' to be 'second-value', got '%s'", secondHeader)
	}
	if firstHeader := w.Header().Get("X-First-Header"); firstHeader != "" {
		t.Errorf("Expected first option header 'X-First-Header' to be overridden, got '%s'", firstHeader)
	}
}

func TestSetHeader_WithDifferentHTTPMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{"X-Method-Header": "method-test"}))(handler)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			if headerValue := w.Header().Get("X-Method-Header"); headerValue != "method-test" {
				t.Errorf("Expected header for %s method to be 'method-test', got '%s'", method, headerValue)
			}
		})
	}
}

func TestDefaultSetHeaderConfig(t *testing.T) {
	cfg := config.DefaultSetHeaderConfig
	if cfg.Headers == nil {
		t.Error("Expected default headers map to be initialized")
	}
	if len(cfg.Headers) != 0 {
		t.Errorf("Expected default headers map to be empty, got %d headers", len(cfg.Headers))
	}
}

func TestSetHeader_HeadersNotAffectedByRequestHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(map[string]string{"X-Response-Header": "response-value"}))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Response-Header", "request-value")
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if responseHeaderValue := w.Header().Get("X-Response-Header"); responseHeaderValue != "response-value" {
		t.Errorf("Expected response header to be 'response-value', got '%s'", responseHeaderValue)
	}
}

func TestSetHeader_LargeNumberOfHeaders(t *testing.T) {
	headers := make(map[string]string)
	for i := range 100 {
		headers[fmt.Sprintf("X-Header-%d", i)] = fmt.Sprintf("value-%d", i)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	middleware := SetHeader(config.WithSetHeaders(headers))(handler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	for i := range 100 {
		expectedKey := fmt.Sprintf("X-Header-%d", i)
		expectedValue := fmt.Sprintf("value-%d", i)
		actualValue := w.Header().Get(expectedKey)
		if actualValue != expectedValue {
			t.Errorf("Expected header '%s' to be '%s', got '%s'", expectedKey, expectedValue, actualValue)
		}
	}
}

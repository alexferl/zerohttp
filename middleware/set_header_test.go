package middleware

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestSetHeader_DefaultConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(SetHeader(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("OK")
	// Content-Length is always present
	if len(w.Header()) > 1 {
		t.Errorf("Expected no custom headers with default config, got %d headers", len(w.Header()))
	}
}

func TestSetHeader_SingleHeader(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		}}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).Header("X-Custom-Header", "custom-value")
}

func TestSetHeader_MultipleHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Custom-Header":  "custom-value",
		"X-Another-Header": "another-value",
		"Cache-Control":    "no-cache",
		"X-API-Version":    "v1.0",
	}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: headers}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	for expectedKey, expectedValue := range headers {
		zhtest.AssertWith(t, w).Header(expectedKey, expectedValue)
	}
}

func TestSetHeader_EmptyHeaderValue(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{
			"X-Empty-Header":  "",
			"X-Normal-Header": "normal-value",
		}}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	emptyHeaderValue := w.Header().Get("X-Empty-Header")
	if emptyHeaderValue != "" {
		t.Errorf("Expected empty header 'X-Empty-Header' to be '', got '%s'", emptyHeaderValue)
	}
	_, exists := w.Header()["X-Empty-Header"]
	if !exists {
		t.Error("Expected empty header 'X-Empty-Header' to exist")
	}
	zhtest.AssertWith(t, w).Header("X-Normal-Header", "normal-value")
}

func TestSetHeader_NilHeaders(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: nil}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
}

func TestSetHeader_OverrideExistingHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		w.Header().Set(httpx.HeaderServer, "Default-Server")
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{
			"Content-Type": "application/json",
			"Server":       "Custom-Server",
		}}),
		handler,
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	// Handler sets headers before middleware, so middleware values should be set
	// but handler writes them first. The SetHeader middleware runs before handler.
	if contentType := w.Header().Get(httpx.HeaderContentType); contentType != "text/html" {
		t.Errorf("Expected Content-Type to be overridden to 'text/html', got '%s'", contentType)
	}
	if server := w.Header().Get(httpx.HeaderServer); server != "Default-Server" {
		t.Errorf("Expected Server to be overridden to 'Default-Server', got '%s'", server)
	}
}

func TestSetHeader_HeadersSetBeforeHandler(t *testing.T) {
	var headerValueInHandler string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerValueInHandler = w.Header().Get("X-Middleware-Header")
		w.WriteHeader(http.StatusOK)
	})
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{
			"X-Middleware-Header": "middleware-value",
		}}),
		handler,
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	if headerValueInHandler != "middleware-value" {
		t.Errorf("Expected header to be visible in handler as 'middleware-value', got '%s'", headerValueInHandler)
	}
	zhtest.AssertWith(t, w).Header("X-Middleware-Header", "middleware-value")
}

func TestSetHeader_CaseInsensitiveHeaders(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{
			"content-type":    "application/json",
			"x-custom-header": "lowercase-key",
		}}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).
		Header(httpx.HeaderContentType, "application/json").
		Header("X-Custom-Header", "lowercase-key")
}

func TestSetHeader_SpecialCharactersInHeaderValue(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{
			"X-Special-Chars": "value with spaces, commas; and: colons",
			"X-Unicode":       "测试值",
			"X-Numbers":       "12345",
		}}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).
		Header("X-Special-Chars", "value with spaces, commas; and: colons").
		Header("X-Unicode", "测试值").
		Header("X-Numbers", "12345")
}

func TestSetHeader_MultipleOptions(t *testing.T) {
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(
			config.SetHeaderConfig{Headers: map[string]string{"X-First-Header": "first-value"}},
			config.SetHeaderConfig{Headers: map[string]string{"X-Second-Header": "second-value"}},
		),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).Header("X-Second-Header", "second-value")
	zhtest.AssertWith(t, w).HeaderNotExists("X-First-Header")
}

func TestSetHeader_WithDifferentHTTPMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := zhtest.NewRequest(method, "/").Build()
			w := zhtest.TestMiddleware(
				SetHeader(config.SetHeaderConfig{Headers: map[string]string{"X-Method-Header": "method-test"}}),
				req,
			)

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
	req := zhtest.NewRequest(http.MethodGet, "/").
		WithHeader("X-Response-Header", "request-value").
		Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: map[string]string{"X-Response-Header": "response-value"}}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).Header("X-Response-Header", "response-value")
}

func TestSetHeader_LargeNumberOfHeaders(t *testing.T) {
	headers := make(map[string]string)
	for i := range 100 {
		headers[fmt.Sprintf("X-Header-%d", i)] = fmt.Sprintf("value-%d", i)
	}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddleware(
		SetHeader(config.SetHeaderConfig{Headers: headers}),
		req,
	)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	for i := range 100 {
		expectedKey := fmt.Sprintf("X-Header-%d", i)
		expectedValue := fmt.Sprintf("value-%d", i)
		if actualValue := w.Header().Get(expectedKey); actualValue != expectedValue {
			t.Errorf("Expected header '%s' to be '%s', got '%s'", expectedKey, expectedValue, actualValue)
		}
	}
}

package middleware

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequestID_ExistingHeader(t *testing.T) {
	handler := &testHandler{}
	existingID := "existing-request-id-123"
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("X-Request-Id", existingID).Build()
	w := zhtest.TestMiddlewareWithHandler(RequestID(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	if handler.requestID != existingID {
		t.Errorf("Expected to use existing request ID %s, got %s", existingID, handler.requestID)
	}
	if reqHeaderValue := handler.request.Header.Get("X-Request-Id"); reqHeaderValue != existingID {
		t.Errorf("Expected request header to be %s, got %s", existingID, reqHeaderValue)
	}
	if respHeaderValue := w.Header().Get("X-Request-Id"); respHeaderValue != existingID {
		t.Errorf("Expected response header to be %s, got %s", existingID, respHeaderValue)
	}
}

func TestRequestID_CustomHeader(t *testing.T) {
	handler := &testHandler{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(RequestID(config.RequestIDConfig{
		Header: "X-Trace-Id",
	}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	reqHeaderValue := handler.request.Header.Get("X-Trace-Id")
	if reqHeaderValue == "" {
		t.Error("Expected custom request header to be set")
	}
	respHeaderValue := w.Header().Get("X-Trace-Id")
	if respHeaderValue == "" {
		t.Error("Expected custom response header to be set")
	}
	if reqHeaderValue != respHeaderValue {
		t.Errorf("Expected request and response headers to match: %s != %s", reqHeaderValue, respHeaderValue)
	}
	zhtest.AssertWith(t, w).HeaderNotExists("X-Request-Id")
}

func TestRequestID_CustomGenerator(t *testing.T) {
	counter := 0
	customIDPrefix := "custom-"
	mw := RequestID(config.RequestIDConfig{
		Generator: func() string {
			counter++
			return customIDPrefix + string(rune('0'+counter))
		},
	})

	handler1 := &testHandler{}
	req1 := zhtest.NewRequest(http.MethodGet, "/").Build()
	w1 := zhtest.TestMiddlewareWithHandler(mw, handler1, req1)

	zhtest.AssertWith(t, w1).Status(http.StatusOK)
	expectedID1 := customIDPrefix + "1"
	if handler1.requestID != expectedID1 {
		t.Errorf("Expected first custom generated ID %s, got %s", expectedID1, handler1.requestID)
	}

	handler2 := &testHandler{}
	req2 := zhtest.NewRequest(http.MethodGet, "/").Build()
	w2 := zhtest.TestMiddlewareWithHandler(mw, handler2, req2)

	zhtest.AssertWith(t, w2).Status(http.StatusOK)
	expectedID2 := customIDPrefix + "2"
	if handler2.requestID != expectedID2 {
		t.Errorf("Expected second custom generated ID %s, got %s", expectedID2, handler2.requestID)
	}
}

func TestRequestID_CustomContextKey(t *testing.T) {
	handler := &testHandler{}
	// Custom context key using a struct type
	type traceKey struct{}
	customKey := traceKey{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(RequestID(config.RequestIDConfig{
		ContextKey: customKey,
	}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	customRequestID := GetRequestID(handler.context, customKey)
	if customRequestID == "" {
		t.Error("Expected request ID to be stored with custom context key")
	}
	if defaultRequestID := GetRequestID(handler.context); defaultRequestID != "" {
		t.Error("Expected request ID not to be available with default key when custom key is used")
	}
	if handler.requestID != "" {
		t.Error("Expected handler's request ID to be empty when custom context key is used")
	}
}

func TestRequestID_EmptyConfigValues(t *testing.T) {
	handler := &testHandler{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(RequestID(config.RequestIDConfig{}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderExists("X-Request-Id")
	if len(handler.requestID) != 32 {
		t.Errorf("Expected default generated ID length 32, got %d", len(handler.requestID))
	}
	if handler.requestID == "" {
		t.Error("Expected request ID to be available with default context key")
	}
}

func TestRequestID_MultipleOptions(t *testing.T) {
	handler := &testHandler{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(RequestID(config.RequestIDConfig{
		Header: "X-Custom-Id",
	}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderExists("X-Custom-Id")
}

func TestGetRequestID_WithCustomKey(t *testing.T) {
	// Custom context key using a struct type
	type myRequestKey struct{}
	customKey := myRequestKey{}
	testRequestID := "test-123"
	ctx := context.WithValue(context.Background(), customKey, testRequestID)
	retrievedID := GetRequestID(ctx, customKey)
	if retrievedID != testRequestID {
		t.Errorf("Expected %s, got %s", testRequestID, retrievedID)
	}
	if defaultID := GetRequestID(ctx); defaultID != "" {
		t.Errorf("Expected empty string with default key, got %s", defaultID)
	}
}

func TestGetRequestID_EdgeCases(t *testing.T) {
	t.Run("no request ID", func(t *testing.T) {
		ctx := context.Background()
		if requestID := GetRequestID(ctx); requestID != "" {
			t.Errorf("Expected empty string, got %s", requestID)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), config.DefaultRequestIDConfig.ContextKey, 123)
		if requestID := GetRequestID(ctx); requestID != "" {
			t.Errorf("Expected empty string for non-string value, got %s", requestID)
		}
	})
}

func TestDefaultRequestIDConfig(t *testing.T) {
	cfg := config.DefaultRequestIDConfig
	if cfg.Header != "X-Request-Id" {
		t.Errorf("Expected default header 'X-Request-Id', got %s", cfg.Header)
	}
	if cfg.Generator == nil {
		t.Error("Expected default generator to be set")
	}
	if cfg.ContextKey != config.RequestIDContextKey {
		t.Errorf("Expected default context key to be RequestIDContextKey, got %v", cfg.ContextKey)
	}
	id := cfg.Generator()
	if len(id) != 32 {
		t.Errorf("Expected default generator to produce 32-char string, got %d", len(id))
	}
	if matched, _ := regexp.MatchString("^[a-f0-9]{32}$", id); !matched {
		t.Errorf("Expected default generator to produce hex string, got %s", id)
	}
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	hexRe := regexp.MustCompile(`^[a-f0-9]{32}$`)
	ids := make(map[string]bool)
	for range 100 {
		id := config.GenerateRequestID()
		if ids[id] {
			t.Errorf("Generated duplicate request ID: %s", id)
		}
		ids[id] = true
		if len(id) != 32 {
			t.Errorf("Expected ID length 32, got %d for ID: %s", len(id), id)
		}
		if !hexRe.MatchString(id) {
			t.Errorf("Expected hex format, got: %s", id)
		}
	}
}

type contextKey string

const existingKey contextKey = "existing_key"

func TestRequestID_PreservesExistingContext(t *testing.T) {
	handler := &testHandler{}
	existingValue := "existing_value"
	ctx := context.WithValue(context.Background(), existingKey, existingValue)
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	req = req.WithContext(ctx)
	w := zhtest.TestMiddlewareWithHandler(RequestID(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	if retrievedValue := handler.context.Value(existingKey); retrievedValue != existingValue {
		t.Errorf("Expected existing context value to be preserved: %v != %v", existingValue, retrievedValue)
	}
	if handler.requestID == "" {
		t.Error("Expected request ID to be available in context")
	}
}

func TestRequestID_CaseInsensitiveHeader(t *testing.T) {
	handler := &testHandler{}
	existingID := "case-test-123"
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("x-request-id", existingID).Build()
	w := zhtest.TestMiddlewareWithHandler(RequestID(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	if handler.requestID != existingID {
		t.Errorf("Expected to use existing request ID %s, got %s", existingID, handler.requestID)
	}
}

// testHandler is a reusable test handler that captures request information
type testHandler struct {
	requestID string
	context   context.Context
	request   *http.Request
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.context = r.Context()
	h.request = r
	h.requestID = GetRequestID(r.Context())
	w.WriteHeader(http.StatusOK)
}

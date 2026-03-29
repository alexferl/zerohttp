package requestid

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestRequestID_ExistingHeader(t *testing.T) {
	handler := &testHandler{}
	existingID := "existing-request-id-123"
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader(httpx.HeaderXRequestId, existingID).Build()
	w := zhtest.TestMiddlewareWithHandler(New(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertEqual(t, existingID, handler.requestID)
	zhtest.AssertEqual(t, existingID, handler.request.Header.Get(httpx.HeaderXRequestId))
	zhtest.AssertEqual(t, existingID, w.Header().Get(httpx.HeaderXRequestId))
}

func TestRequestID_CustomHeader(t *testing.T) {
	handler := &testHandler{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(New(Config{
		Header: "X-Trace-Id",
	}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	reqHeaderValue := handler.request.Header.Get("X-Trace-Id")
	zhtest.AssertNotEmpty(t, reqHeaderValue)
	respHeaderValue := w.Header().Get("X-Trace-Id")
	zhtest.AssertNotEmpty(t, respHeaderValue)
	zhtest.AssertEqual(t, reqHeaderValue, respHeaderValue)
	zhtest.AssertWith(t, w).HeaderNotExists(httpx.HeaderXRequestId)
}

func TestRequestID_CustomGenerator(t *testing.T) {
	counter := 0
	customIDPrefix := "custom-"
	mw := New(Config{
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
	zhtest.AssertEqual(t, expectedID1, handler1.requestID)

	handler2 := &testHandler{}
	req2 := zhtest.NewRequest(http.MethodGet, "/").Build()
	w2 := zhtest.TestMiddlewareWithHandler(mw, handler2, req2)

	zhtest.AssertWith(t, w2).Status(http.StatusOK)
	expectedID2 := customIDPrefix + "2"
	zhtest.AssertEqual(t, expectedID2, handler2.requestID)
}

func TestRequestID_CustomContextKey(t *testing.T) {
	handler := &testHandler{}
	// Custom context key using a struct type
	type traceKey struct{}
	customKey := traceKey{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(New(Config{
		ContextKey: customKey,
	}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	customRequestID := Get(handler.context, customKey)
	zhtest.AssertNotEmpty(t, customRequestID)
	zhtest.AssertEmpty(t, Get(handler.context))
	zhtest.AssertEmpty(t, handler.requestID)
}

func TestRequestID_EmptyConfigValues(t *testing.T) {
	handler := &testHandler{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(New(Config{}), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertWith(t, w).HeaderExists(httpx.HeaderXRequestId)
	zhtest.AssertEqual(t, 32, len(handler.requestID))
	zhtest.AssertNotEmpty(t, handler.requestID)
}

func TestRequestID_MultipleOptions(t *testing.T) {
	handler := &testHandler{}
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.TestMiddlewareWithHandler(New(Config{
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
	retrievedID := Get(ctx, customKey)
	zhtest.AssertEqual(t, testRequestID, retrievedID)
	zhtest.AssertEmpty(t, Get(ctx))
}

func TestGetRequestID_EdgeCases(t *testing.T) {
	t.Run("no request ID", func(t *testing.T) {
		ctx := context.Background()
		zhtest.AssertEmpty(t, Get(ctx))
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), DefaultConfig.ContextKey, 123)
		zhtest.AssertEmpty(t, Get(ctx))
	})
}

func TestDefaultRequestIDConfig(t *testing.T) {
	cfg := DefaultConfig
	zhtest.AssertEqual(t, httpx.HeaderXRequestId, cfg.Header)
	zhtest.AssertNotNil(t, cfg.Generator)
	zhtest.AssertEqual(t, ContextKey, cfg.ContextKey)
	id := cfg.Generator()
	zhtest.AssertEqual(t, 32, len(id))
	matched, _ := regexp.MatchString("^[a-f0-9]{32}$", id)
	zhtest.AssertTrue(t, matched)
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	hexRe := regexp.MustCompile(`^[a-f0-9]{32}$`)
	ids := make(map[string]bool)
	for range 100 {
		id := GenerateRequestID()
		zhtest.AssertFalse(t, ids[id])
		ids[id] = true
		zhtest.AssertEqual(t, 32, len(id))
		zhtest.AssertTrue(t, hexRe.MatchString(id))
	}
}

type testContextKey string

const existingKey testContextKey = "existing_key"

func TestRequestID_PreservesExistingContext(t *testing.T) {
	handler := &testHandler{}
	existingValue := "existing_value"
	ctx := context.WithValue(context.Background(), existingKey, existingValue)
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	req = req.WithContext(ctx)
	w := zhtest.TestMiddlewareWithHandler(New(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertEqual(t, existingValue, handler.context.Value(existingKey))
	zhtest.AssertNotEmpty(t, handler.requestID)
}

func TestRequestID_CaseInsensitiveHeader(t *testing.T) {
	handler := &testHandler{}
	existingID := "case-test-123"
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("x-request-id", existingID).Build()
	w := zhtest.TestMiddlewareWithHandler(New(), handler, req)

	zhtest.AssertWith(t, w).Status(http.StatusOK)
	zhtest.AssertEqual(t, existingID, handler.requestID)
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
	h.requestID = Get(r.Context())
	w.WriteHeader(http.StatusOK)
}

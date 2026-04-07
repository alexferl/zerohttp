package recover

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/metrics"
	"github.com/alexferl/zerohttp/zhtest"
)

type mockLogger struct {
	debugLogs   []string
	infoLogs    []string
	errorLogs   []string
	errorFields [][]log.Field
}

func (m *mockLogger) Debug(msg string, fields ...log.Field) { m.debugLogs = append(m.debugLogs, msg) }
func (m *mockLogger) Info(msg string, fields ...log.Field)  { m.infoLogs = append(m.infoLogs, msg) }
func (m *mockLogger) Warn(msg string, fields ...log.Field)  {}
func (m *mockLogger) Error(msg string, fields ...log.Field) {
	m.errorLogs = append(m.errorLogs, msg)
	m.errorFields = append(m.errorFields, fields)
}
func (m *mockLogger) Panic(msg string, fields ...log.Field)      {}
func (m *mockLogger) Fatal(msg string, fields ...log.Field)      {}
func (m *mockLogger) WithFields(fields ...log.Field) log.Logger  { return m }
func (m *mockLogger) WithContext(ctx context.Context) log.Logger { return m }

func panicHandler(panicValue any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(panicValue)
	})
}

func normalHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

func TestRecover_NoPanic(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger)(normalHandler())
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 0, len(logger.errorLogs))
	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("OK")
}

func TestRecover_WithPanic(t *testing.T) {
	logger := &mockLogger{}
	panicMsg := "test panic"
	handler := New(logger)(panicHandler(panicMsg))
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("X-Request-Id", "test-req-123").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))
	zhtest.AssertEqual(t, "Recovered from panic", logger.errorLogs[0])
	zhtest.AssertWith(t, w).Status(http.StatusInternalServerError)

	fields := logger.errorFields[0]
	foundPanic, foundRequestID, foundStack := false, false, false
	for _, field := range fields {
		switch field.Key {
		case "panic":
			if field.Value == panicMsg {
				foundPanic = true
			}
		case "request_id":
			if field.Value == "test-req-123" {
				foundRequestID = true
			}
		case "stack":
			if stackStr, ok := field.Value.(string); ok && len(stackStr) > 0 {
				foundStack = true
			}
		}
	}
	zhtest.AssertTrue(t, foundPanic)
	zhtest.AssertTrue(t, foundRequestID)
	zhtest.AssertTrue(t, foundStack)
}

func TestRecover_HTTPAbortHandler(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger)(panicHandler(http.ErrAbortHandler))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	defer func() {
		r := recover()
		zhtest.AssertEqual(t, http.ErrAbortHandler, r)
	}()
	zhtest.Serve(handler, req)
	zhtest.AssertFail(t, "Expected panic to be re-raised")
}

func TestRecover_UpgradeConnection(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger)(panicHandler("websocket panic"))
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("Connection", "Upgrade").Build()
	w := zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, http.StatusOK, w.Code)
	zhtest.AssertEqual(t, 1, len(logger.errorLogs))
}

func TestRecover_CustomConfig(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger, Config{
		StackSize:        1024,
		EnableStackTrace: config.Bool(true),
	})(panicHandler("custom config panic"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	fields := logger.errorFields[0]
	foundStack := false
	for _, field := range fields {
		if field.Key == "stack" {
			if stackStr, ok := field.Value.(string); ok && len(stackStr) > 0 {
				foundStack = true
			}
		}
	}
	zhtest.AssertTrue(t, foundStack)
}

func TestRecover_DisabledStackTrace(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger, Config{
		StackSize:        4096,
		EnableStackTrace: config.Bool(false),
	})(panicHandler("no stack panic"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	fields := logger.errorFields[0]
	for _, field := range fields {
		if field.Key == "stack" {
			zhtest.AssertFail(t, "Did not expect stack trace to be logged when disabled")
		}
	}
}

func TestRecover_InvalidStackSize(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger, Config{
		StackSize:        0,
		EnableStackTrace: config.Bool(true),
	})(panicHandler("invalid stack size"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	fields := logger.errorFields[0]
	foundStack := false
	for _, field := range fields {
		if field.Key == "stack" {
			if stackStr, ok := field.Value.(string); ok && len(stackStr) > 0 {
				foundStack = true
			}
		}
	}
	zhtest.AssertTrue(t, foundStack)
}

func TestRecover_ErrorValue(t *testing.T) {
	logger := &mockLogger{}
	testError := errors.New("test error panic")
	handler := New(logger)(panicHandler(testError))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	fields := logger.errorFields[0]
	foundError := false
	for _, field := range fields {
		if field.Key == "panic" && field.Value == testError {
			foundError = true
		}
	}
	zhtest.AssertTrue(t, foundError)
}

func TestRecover_StringPanic(t *testing.T) {
	logger := &mockLogger{}
	panicMsg := "string panic message"
	handler := New(logger)(panicHandler(panicMsg))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	fields := logger.errorFields[0]
	foundPanic := false
	for _, field := range fields {
		if field.Key == "panic" && field.Value == panicMsg {
			foundPanic = true
		}
	}
	zhtest.AssertTrue(t, foundPanic)
}

func TestDefaultRecoverConfig(t *testing.T) {
	cfg := DefaultConfig
	expectedStackSize := int64(4 << 10)
	zhtest.AssertEqual(t, expectedStackSize, cfg.StackSize)
	zhtest.AssertTrue(t, *cfg.EnableStackTrace)
}

func TestRecover_MultipleOptions(t *testing.T) {
	logger := &mockLogger{}
	handler := New(logger, Config{
		StackSize:        1024,
		EnableStackTrace: config.Bool(true),
	})(panicHandler("multiple options"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	fields := logger.errorFields[0]
	foundStack := false
	for _, field := range fields {
		if field.Key == "stack" {
			foundStack = true
		}
	}
	zhtest.AssertTrue(t, foundStack)
}

func TestRecover_PanicFieldLogging(t *testing.T) {
	logger := &mockLogger{}
	panicValue := "test panic for field check"
	handler := New(logger)(panicHandler(panicValue))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	zhtest.AssertTrue(t, len(logger.errorFields) > 0)

	fields := logger.errorFields[0]
	foundPanicField := false
	for _, field := range fields {
		if field.Key == "panic" && field.Value == panicValue {
			foundPanicField = true
			break
		}
	}
	zhtest.AssertTrue(t, foundPanicField)
}

func TestRecover_NonHandlerError(t *testing.T) {
	logger := &mockLogger{}
	// Regular panic (not a handler error wrapper)
	handler := New(logger)(panicHandler("random panic"))

	// Test JSON response with Accept header
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("Accept", "application/json").Build()
	w := zhtest.Serve(handler, req)

	// Should return 500
	zhtest.AssertEqual(t, http.StatusInternalServerError, w.Code)

	// Should log as error with stack trace
	zhtest.AssertEqual(t, 1, len(logger.errorLogs))

	// Should return problem detail response body
	contentType := w.Header().Get(httpx.HeaderContentType)
	zhtest.AssertTrue(t, strings.Contains(contentType, "application/problem+json"))

	body := w.Body.String()
	zhtest.AssertTrue(t, strings.Contains(body, `"status":500`))
	zhtest.AssertTrue(t, strings.Contains(body, `"title":"Internal Server Error"`))

	// Test JSON response without Accept header (defaults to JSON)
	logger = &mockLogger{} // Reset logger
	handler = New(logger)(panicHandler("random panic"))
	req = zhtest.NewRequest(http.MethodGet, "/").Build()
	w = zhtest.Serve(handler, req)

	contentType = w.Header().Get(httpx.HeaderContentType)
	zhtest.AssertTrue(t, strings.Contains(contentType, "application/problem+json"))
}

func TestRecover_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := &mockLogger{}
	mw := New(logger)

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, metrics.Config{
		Enabled:       config.Bool(true),
		PathLabelFunc: func(p string) string { return p },
	})
	wrapped := metricsMw(mw(panicHandler("test panic")))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	zhtest.AssertEqual(t, http.StatusInternalServerError, rr.Code)

	// Check metrics
	families := reg.Gather()
	var counter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "recover_panics_total" {
			counter = &f
			break
		}
	}
	zhtest.AssertNotNil(t, counter)
	zhtest.AssertEqual(t, 1, len(counter.Metrics))
}

func TestRecover_CustomRequestIDHeader(t *testing.T) {
	logger := &mockLogger{}
	customHeader := "X-Custom-Request-ID"

	cfg := Config{
		RequestIDHeader:  customHeader,
		EnableStackTrace: config.Bool(false),
	}

	handler := New(logger, cfg)(panicHandler("test panic"))

	// Test 1: Request with custom header should log that request ID
	req1 := zhtest.NewRequest(http.MethodGet, "/").WithHeader(customHeader, "custom-req-456").Build()
	w1 := zhtest.Serve(handler, req1)

	zhtest.AssertEqual(t, http.StatusInternalServerError, w1.Code)

	foundCustomID := false
	for _, fields := range logger.errorFields {
		for _, field := range fields {
			if field.Key == "request_id" && field.Value == "custom-req-456" {
				foundCustomID = true
				break
			}
		}
	}
	zhtest.AssertTrue(t, foundCustomID)

	// Reset logger for next test
	logger.errorFields = nil
	logger.errorLogs = nil

	// Test 2: Request with default X-Request-Id header should NOT use it
	req2 := zhtest.NewRequest(http.MethodGet, "/").WithHeader("X-Request-Id", "should-be-ignored").Build()
	w2 := zhtest.Serve(handler, req2)

	zhtest.AssertEqual(t, http.StatusInternalServerError, w2.Code)

	foundDefaultID := false
	for _, fields := range logger.errorFields {
		for _, field := range fields {
			if field.Key == "request_id" && field.Value == "should-be-ignored" {
				foundDefaultID = true
				break
			}
		}
	}
	zhtest.AssertFalse(t, foundDefaultID)
}

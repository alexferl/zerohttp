package middleware

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
	handler := Recover(logger)(normalHandler())
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	w := zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 0 {
		t.Errorf("Expected no error logs, got %d", len(logger.errorLogs))
	}
	zhtest.AssertWith(t, w).Status(http.StatusOK).Body("OK")
}

func TestRecover_WithPanic(t *testing.T) {
	logger := &mockLogger{}
	panicMsg := "test panic"
	handler := Recover(logger)(panicHandler(panicMsg))
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("X-Request-Id", "test-req-123").Build()
	w := zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	if logger.errorLogs[0] != "Recovered from panic" {
		t.Errorf("Expected message 'Recovered from panic', got %s", logger.errorLogs[0])
	}
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
	if !foundPanic {
		t.Error("Expected panic value to be logged")
	}
	if !foundRequestID {
		t.Error("Expected request ID to be logged")
	}
	if !foundStack {
		t.Error("Expected stack trace to be logged")
	}
}

func TestRecover_HTTPAbortHandler(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger)(panicHandler(http.ErrAbortHandler))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	defer func() {
		if r := recover(); r != http.ErrAbortHandler {
			t.Errorf("Expected http.ErrAbortHandler to be re-panicked, got %v", r)
		}
	}()
	zhtest.Serve(handler, req)
	t.Error("Expected panic to be re-raised")
}

func TestRecover_UpgradeConnection(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger)(panicHandler("websocket panic"))
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("Connection", "Upgrade").Build()
	w := zhtest.Serve(handler, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected no status code change for upgrade connection, got %d", w.Code)
	}
	if len(logger.errorLogs) != 1 {
		t.Errorf("Expected panic to be logged even for upgrade connections")
	}
}

func TestRecover_CustomConfig(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger, config.RecoverConfig{
		StackSize:        1024,
		EnableStackTrace: true,
	})(panicHandler("custom config panic"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	fields := logger.errorFields[0]
	foundStack := false
	for _, field := range fields {
		if field.Key == "stack" {
			if stackStr, ok := field.Value.(string); ok && len(stackStr) > 0 {
				foundStack = true
			}
		}
	}
	if !foundStack {
		t.Error("Expected stack trace to be present with custom config")
	}
}

func TestRecover_DisabledStackTrace(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger, config.RecoverConfig{
		StackSize:        4096,
		EnableStackTrace: false,
	})(panicHandler("no stack panic"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	fields := logger.errorFields[0]
	for _, field := range fields {
		if field.Key == "stack" {
			t.Error("Did not expect stack trace to be logged when disabled")
		}
	}
}

func TestRecover_InvalidStackSize(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger, config.RecoverConfig{
		StackSize:        0,
		EnableStackTrace: true,
	})(panicHandler("invalid stack size"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	fields := logger.errorFields[0]
	foundStack := false
	for _, field := range fields {
		if field.Key == "stack" {
			if stackStr, ok := field.Value.(string); ok && len(stackStr) > 0 {
				foundStack = true
			}
		}
	}
	if !foundStack {
		t.Error("Expected stack trace to be present even with invalid config")
	}
}

func TestRecover_ErrorValue(t *testing.T) {
	logger := &mockLogger{}
	testError := errors.New("test error panic")
	handler := Recover(logger)(panicHandler(testError))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	fields := logger.errorFields[0]
	foundError := false
	for _, field := range fields {
		if field.Key == "panic" && field.Value == testError {
			foundError = true
		}
	}
	if !foundError {
		t.Error("Expected error value to be logged correctly")
	}
}

func TestRecover_StringPanic(t *testing.T) {
	logger := &mockLogger{}
	panicMsg := "string panic message"
	handler := Recover(logger)(panicHandler(panicMsg))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	fields := logger.errorFields[0]
	foundPanic := false
	for _, field := range fields {
		if field.Key == "panic" && field.Value == panicMsg {
			foundPanic = true
		}
	}
	if !foundPanic {
		t.Error("Expected string panic value to be logged")
	}
}

func TestDefaultRecoverConfig(t *testing.T) {
	cfg := config.DefaultRecoverConfig
	expectedStackSize := int64(4 << 10)
	if cfg.StackSize != expectedStackSize {
		t.Errorf("Expected default stack size %d, got %d", expectedStackSize, cfg.StackSize)
	}
	if !cfg.EnableStackTrace {
		t.Error("Expected default EnableStackTrace to be true")
	}
}

func TestRecover_MultipleOptions(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger, config.RecoverConfig{
		StackSize:        1024,
		EnableStackTrace: true,
	})(panicHandler("multiple options"))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	fields := logger.errorFields[0]
	foundStack := false
	for _, field := range fields {
		if field.Key == "stack" {
			foundStack = true
		}
	}
	if !foundStack {
		t.Error("Expected stack trace to be present (should use options)")
	}
}

func TestRecover_PanicFieldLogging(t *testing.T) {
	logger := &mockLogger{}
	panicValue := "test panic for field check"
	handler := Recover(logger)(panicHandler(panicValue))
	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	zhtest.Serve(handler, req)

	if len(logger.errorFields) == 0 {
		t.Fatal("Expected error fields to be captured")
	}
	fields := logger.errorFields[0]
	foundPanicField := false
	for _, field := range fields {
		if field.Key == "panic" && field.Value == panicValue {
			foundPanicField = true
			break
		}
	}
	if !foundPanicField {
		t.Error("Expected panic field to be logged using log.P() helper")
	}
}

func TestRecover_NonHandlerError(t *testing.T) {
	logger := &mockLogger{}
	// Regular panic (not a handler error wrapper)
	handler := Recover(logger)(panicHandler("random panic"))

	// Test JSON response with Accept header
	req := zhtest.NewRequest(http.MethodGet, "/").WithHeader("Accept", "application/json").Build()
	w := zhtest.Serve(handler, req)

	// Should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 Internal Server Error, got %d", w.Code)
	}

	// Should log as error with stack trace
	if len(logger.errorLogs) != 1 {
		t.Errorf("Expected 1 error log for panics, got %d", len(logger.errorLogs))
	}

	// Should return problem detail response body
	contentType := w.Header().Get(httpx.HeaderContentType)
	if !strings.Contains(contentType, "application/problem+json") {
		t.Errorf("Expected Content-Type to contain application/problem+json, got %s", contentType)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":500`) {
		t.Errorf("Expected body to contain status 500, got %s", body)
	}
	if !strings.Contains(body, `"title":"Internal Server Error"`) {
		t.Errorf("Expected body to contain title 'Internal Server Error', got %s", body)
	}

	// Test plain text response without Accept header
	logger = &mockLogger{} // Reset logger
	handler = Recover(logger)(panicHandler("random panic"))
	req = zhtest.NewRequest(http.MethodGet, "/").Build()
	w = zhtest.Serve(handler, req)

	contentType = w.Header().Get(httpx.HeaderContentType)
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to contain text/plain, got %s", contentType)
	}
}

func TestRecover_Metrics(t *testing.T) {
	reg := metrics.NewRegistry()
	logger := &mockLogger{}
	mw := Recover(logger)

	// Wrap with metrics middleware to provide registry in context
	metricsMw := metrics.NewMiddleware(reg, config.MetricsConfig{
		Enabled:       true,
		PathLabelFunc: func(p string) string { return p },
	})
	wrapped := metricsMw(mw(panicHandler("test panic")))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	// Check metrics
	families := reg.Gather()
	var counter *metrics.MetricFamily
	for _, f := range families {
		if f.Name == "recover_panics_total" {
			counter = &f
			break
		}
	}
	if counter == nil {
		t.Fatal("expected recover_panics_total metric")
	}
	if len(counter.Metrics) != 1 {
		t.Errorf("expected 1 panic metric, got %d", len(counter.Metrics))
	}
}

func TestRecover_CustomRequestIDHeader(t *testing.T) {
	logger := &mockLogger{}
	customHeader := "X-Custom-Request-ID"

	cfg := config.RecoverConfig{
		RequestIDHeader:  customHeader,
		EnableStackTrace: false,
	}

	handler := Recover(logger, cfg)(panicHandler("test panic"))

	// Test 1: Request with custom header should log that request ID
	req1 := zhtest.NewRequest(http.MethodGet, "/").WithHeader(customHeader, "custom-req-456").Build()
	w1 := zhtest.Serve(handler, req1)

	if w1.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w1.Code)
	}

	foundCustomID := false
	for _, fields := range logger.errorFields {
		for _, field := range fields {
			if field.Key == "request_id" && field.Value == "custom-req-456" {
				foundCustomID = true
				break
			}
		}
	}
	if !foundCustomID {
		t.Error("expected custom request ID to be logged")
	}

	// Reset logger for next test
	logger.errorFields = nil
	logger.errorLogs = nil

	// Test 2: Request with default X-Request-Id header should NOT use it
	req2 := zhtest.NewRequest(http.MethodGet, "/").WithHeader("X-Request-Id", "should-be-ignored").Build()
	w2 := zhtest.Serve(handler, req2)

	if w2.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w2.Code)
	}

	foundDefaultID := false
	for _, fields := range logger.errorFields {
		for _, field := range fields {
			if field.Key == "request_id" && field.Value == "should-be-ignored" {
				foundDefaultID = true
				break
			}
		}
	}
	if foundDefaultID {
		t.Error("expected default X-Request-Id header to be ignored when custom header is configured")
	}
}

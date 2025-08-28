package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
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
		if _, err := w.Write([]byte("OK")); err != nil {
			panic(fmt.Errorf("failed to write test response: %w", err))
		}
	})
}

func TestRecover_NoPanic(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger)(normalHandler())
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(logger.errorLogs) != 0 {
		t.Errorf("Expected no error logs, got %d", len(logger.errorLogs))
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

func TestRecover_WithPanic(t *testing.T) {
	logger := &mockLogger{}
	panicMsg := "test panic"
	handler := Recover(logger)(panicHandler(panicMsg))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "test-req-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(logger.errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
	}
	if logger.errorLogs[0] != "Recovered from panic" {
		t.Errorf("Expected message 'Recovered from panic', got %s", logger.errorLogs[0])
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
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
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != http.ErrAbortHandler {
			t.Errorf("Expected http.ErrAbortHandler to be re-panicked, got %v", r)
		}
	}()
	handler.ServeHTTP(w, req)
	t.Error("Expected panic to be re-raised")
}

func TestRecover_UpgradeConnection(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger)(panicHandler("websocket panic"))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Connection", "Upgrade")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected no status code change for upgrade connection, got %d", w.Code)
	}
	if len(logger.errorLogs) != 1 {
		t.Errorf("Expected panic to be logged even for upgrade connections")
	}
}

func TestRecover_CustomConfig(t *testing.T) {
	logger := &mockLogger{}
	handler := Recover(logger,
		config.WithRecoverStackSize(1024),
		config.WithRecoverEnableStackTrace(true),
	)(panicHandler("custom config panic"))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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
	handler := Recover(logger,
		config.WithRecoverStackSize(4096),
		config.WithRecoverEnableStackTrace(false),
	)(panicHandler("no stack panic"))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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
	handler := Recover(logger,
		config.WithRecoverStackSize(0),
		config.WithRecoverEnableStackTrace(true),
	)(panicHandler("invalid stack size"))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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
	handler := Recover(logger,
		config.WithRecoverStackSize(1024),
		config.WithRecoverEnableStackTrace(true),
	)(panicHandler("multiple options"))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

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

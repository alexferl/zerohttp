package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/zhtest"
)

type logEntry struct {
	message string
	fields  []log.Field
}

type requestLoggerMockLogger struct {
	debugLogs []logEntry
	infoLogs  []logEntry
	warnLogs  []logEntry
	errorLogs []logEntry
}

func (m *requestLoggerMockLogger) Debug(msg string, fields ...log.Field) {
	m.debugLogs = append(m.debugLogs, logEntry{msg, fields})
}

func (m *requestLoggerMockLogger) Info(msg string, fields ...log.Field) {
	m.infoLogs = append(m.infoLogs, logEntry{msg, fields})
}

func (m *requestLoggerMockLogger) Warn(msg string, fields ...log.Field) {
	m.warnLogs = append(m.warnLogs, logEntry{msg, fields})
}

func (m *requestLoggerMockLogger) Error(msg string, fields ...log.Field) {
	m.errorLogs = append(m.errorLogs, logEntry{msg, fields})
}
func (m *requestLoggerMockLogger) Panic(msg string, fields ...log.Field)      {}
func (m *requestLoggerMockLogger) Fatal(msg string, fields ...log.Field)      {}
func (m *requestLoggerMockLogger) WithFields(fields ...log.Field) log.Logger  { return m }
func (m *requestLoggerMockLogger) WithContext(ctx context.Context) log.Logger { return m }

func findFieldValue(fields []log.Field, key string) (any, bool) {
	for _, field := range fields {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

type statusTestHandler struct {
	statusCode int
	delay      time.Duration
}

func (h *statusTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.delay > 0 {
		time.Sleep(h.delay)
	}
	if h.statusCode > 0 {
		w.WriteHeader(h.statusCode)
	}
	_, _ = w.Write([]byte("test response"))
}

func TestRequestLogger_LogLevels(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	middleware := RequestLogger(logger)

	t.Run("client error", func(t *testing.T) {
		handler := &statusTestHandler{statusCode: http.StatusNotFound}
		req := zhtest.NewRequest(http.MethodGet, "/notfound").Build()
		zhtest.TestMiddlewareWithHandler(middleware, handler, req)

		if len(logger.warnLogs) != 1 {
			t.Fatalf("Expected 1 warn log, got %d", len(logger.warnLogs))
		}
		if value, found := findFieldValue(logger.warnLogs[0].fields, "status"); !found || value != http.StatusNotFound {
			t.Errorf("Expected status 404, got %v", value)
		}
	})

	t.Run("server error", func(t *testing.T) {
		handler := &statusTestHandler{statusCode: http.StatusInternalServerError}
		req := zhtest.NewRequest(http.MethodGet, "/error").Build()
		zhtest.TestMiddlewareWithHandler(middleware, handler, req)

		if len(logger.errorLogs) != 1 {
			t.Fatalf("Expected 1 error log, got %d", len(logger.errorLogs))
		}
		if value, found := findFieldValue(logger.errorLogs[0].fields, "status"); !found || value != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %v", value)
		}
	})

	t.Run("logErrors disabled", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusInternalServerError}
		middleware := RequestLogger(logger, config.WithRequestLoggerLogErrors(false))

		req := zhtest.NewRequest(http.MethodGet, "/error").Build()
		zhtest.TestMiddlewareWithHandler(middleware, handler, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}
		if len(logger.errorLogs) != 0 {
			t.Errorf("Expected no error logs when LogErrors is false, got %d", len(logger.errorLogs))
		}
	})
}

func TestRequestLogger_FieldsAndDurations(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	delay := 10 * time.Millisecond
	handler := &statusTestHandler{statusCode: http.StatusOK, delay: delay}
	middleware := RequestLogger(logger)(handler)

	req := zhtest.NewRequest(http.MethodGet, "/slow").Build()
	start := time.Now()
	zhtest.Serve(middleware, req)
	elapsed := time.Since(start)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}

	entry := logger.infoLogs[0]
	if value, found := findFieldValue(entry.fields, "duration_ns"); !found {
		t.Error("Expected duration_ns field to be present")
	} else {
		durationNS, ok := value.(int64)
		if !ok {
			t.Errorf("Expected duration_ns to be int64, got %T", value)
		} else if time.Duration(durationNS) < delay {
			t.Errorf("Expected duration to be at least %v, got %v", delay, time.Duration(durationNS))
		} else if time.Duration(durationNS) > elapsed+time.Millisecond {
			t.Errorf("Expected duration to be less than %v, got %v", elapsed+time.Millisecond, time.Duration(durationNS))
		}
	}
	if value, found := findFieldValue(entry.fields, "duration_human"); !found {
		t.Error("Expected duration_human field to be present")
	} else if durationStr, ok := value.(string); !ok || !strings.Contains(durationStr, "ms") && !strings.Contains(durationStr, "µs") && !strings.Contains(durationStr, "ns") {
		t.Errorf("Expected duration_human to contain time unit, got %s", value)
	}
}

func TestRequestLogger_EmptyPath(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger)(handler)

	req := zhtest.NewRequest(http.MethodGet, "/").Build()
	req.URL.Path = ""
	zhtest.Serve(middleware, req)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}
	entry := logger.infoLogs[0]
	if value, found := findFieldValue(entry.fields, "path"); !found || value != "/" {
		t.Errorf("Expected empty path to be '/', got %v", value)
	}
}

func TestRequestLogger_RequestIDField(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger)(handler)

	req := zhtest.NewRequest(http.MethodGet, "/test").WithHeader("X-Request-Id", "test-123").Build()
	zhtest.Serve(middleware, req)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}

	entry := logger.infoLogs[0]
	if value, found := findFieldValue(entry.fields, "request_id"); !found || value != "test-123" {
		t.Errorf("Expected request_id 'test-123', got %v", value)
	}
}

func TestRequestLogger_NoRequestIDField(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger)(handler)

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(middleware, req)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}

	entry := logger.infoLogs[0]
	if _, found := findFieldValue(entry.fields, "request_id"); found {
		t.Error("Expected request_id field not to be present when header is missing")
	}
}

func TestRequestLogger_CustomFields(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger,
		config.WithRequestLoggerFields([]config.LogField{
			config.FieldMethod, config.FieldPath, config.FieldStatus,
		}),
	)(handler)

	req := zhtest.NewRequest(http.MethodPost, "/api/users").Build()
	zhtest.Serve(middleware, req)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}

	entry := logger.infoLogs[0]
	expectedFields := []string{"method", "path", "status"}
	unexpectedFields := []string{"uri", "host", "protocol", "user_agent"}

	for _, field := range expectedFields {
		if _, found := findFieldValue(entry.fields, field); !found {
			t.Errorf("Expected field %s to be present", field)
		}
	}
	for _, field := range unexpectedFields {
		if _, found := findFieldValue(entry.fields, field); found {
			t.Errorf("Expected field %s not to be present", field)
		}
	}
}

func TestRequestLogger_ExemptPaths(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger, config.WithRequestLoggerExemptPaths([]string{"/health", "/metrics"}))(handler)

	req1 := zhtest.NewRequest(http.MethodGet, "/health").Build()
	zhtest.Serve(middleware, req1)

	if len(logger.infoLogs) != 0 {
		t.Errorf("Expected no logs for exempt path, got %d", len(logger.infoLogs))
	}

	req2 := zhtest.NewRequest(http.MethodGet, "/api").Build()
	zhtest.Serve(middleware, req2)

	if len(logger.infoLogs) != 1 {
		t.Errorf("Expected 1 log for non-exempt path, got %d", len(logger.infoLogs))
	}
}

func TestRequestLogger_NilFields(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger, config.WithRequestLoggerFields(nil))(handler)

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(middleware, req)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}
	entry := logger.infoLogs[0]
	if _, found := findFieldValue(entry.fields, "method"); !found {
		t.Error("Expected method field to be present with nil Fields config")
	}
}

func TestRequestLogger_MultipleOptions(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := RequestLogger(logger,
		config.WithRequestLoggerFields([]config.LogField{config.FieldMethod}),
		config.WithRequestLoggerFields([]config.LogField{config.FieldPath}),
	)(handler)

	req := zhtest.NewRequest(http.MethodGet, "/test").Build()
	zhtest.Serve(middleware, req)

	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}
	entry := logger.infoLogs[0]
	if _, found := findFieldValue(entry.fields, "method"); found {
		t.Error("Expected method field not to be present (overridden by second option)")
	}
	if _, found := findFieldValue(entry.fields, "path"); !found {
		t.Error("Expected path field from last option to be present")
	}
}

func TestDefaultRequestLoggerConfig(t *testing.T) {
	cfg := config.DefaultRequestLoggerConfig
	if !cfg.LogErrors {
		t.Error("Expected default LogErrors to be true")
	}
	expectedFieldCount := 13
	if len(cfg.Fields) != expectedFieldCount {
		t.Errorf("Expected %d default fields, got %d", expectedFieldCount, len(cfg.Fields))
	}
	expectedFields := []config.LogField{
		config.FieldMethod, config.FieldURI, config.FieldPath, config.FieldHost, config.FieldProtocol,
		config.FieldReferer, config.FieldUserAgent, config.FieldStatus, config.FieldDurationNS,
		config.FieldDurationHuman, config.FieldRemoteAddr, config.FieldClientIP, config.FieldRequestID,
	}
	fieldMap := make(map[config.LogField]bool)
	for _, field := range cfg.Fields {
		fieldMap[field] = true
	}
	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("Expected field %s to be in default cfg", expected)
		}
	}
	if len(cfg.ExemptPaths) != 0 {
		t.Errorf("Expected default exempt paths to be empty, got %d", len(cfg.ExemptPaths))
	}
}

func TestResponseWriter_MultipleWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK, headerWritten: false}
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", rw.statusCode)
	}
	rw.WriteHeader(http.StatusInternalServerError)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected status code to remain 404, got %d", rw.statusCode)
	}
}

func TestResponseWriter_WriteWithoutHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK, headerWritten: false}
	_, err := rw.Write([]byte("test"))
	if err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rw.statusCode)
	}
	if !rw.headerWritten {
		t.Error("Expected headerWritten to be true after Write")
	}
}

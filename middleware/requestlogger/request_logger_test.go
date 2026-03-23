package requestlogger

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/internal/rwutil"
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
	middleware := New(logger)

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
		middleware := New(logger, Config{LogErrors: false})

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
	middleware := New(logger)(handler)

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
	middleware := New(logger)(handler)

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
	middleware := New(logger)(handler)

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
	middleware := New(logger)(handler)

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
	middleware := New(logger, Config{
		Fields: []LogField{
			FieldMethod, FieldPath, FieldStatus,
		},
	})(handler)

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

func TestRequestLogger_ExcludedPaths(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := New(logger, Config{ExcludedPaths: []string{"/health", "/metrics"}})(handler)

	req1 := zhtest.NewRequest(http.MethodGet, "/health").Build()
	zhtest.Serve(middleware, req1)

	if len(logger.infoLogs) != 0 {
		t.Errorf("Expected no logs for excluded path, got %d", len(logger.infoLogs))
	}

	req2 := zhtest.NewRequest(http.MethodGet, "/api").Build()
	zhtest.Serve(middleware, req2)

	if len(logger.infoLogs) != 1 {
		t.Errorf("Expected 1 log for non-excluded path, got %d", len(logger.infoLogs))
	}
}

func TestRequestLogger_NilFields(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := &statusTestHandler{statusCode: http.StatusOK}
	middleware := New(logger, Config{Fields: nil})(handler)

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
	middleware := New(logger, Config{
		Fields: []LogField{FieldPath},
	})(handler)

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
	cfg := DefaultConfig
	if !cfg.LogErrors {
		t.Error("Expected default LogErrors to be true")
	}
	expectedFieldCount := 13
	if len(cfg.Fields) != expectedFieldCount {
		t.Errorf("Expected %d default fields, got %d", expectedFieldCount, len(cfg.Fields))
	}
	expectedFields := []LogField{
		FieldMethod, FieldURI, FieldPath, FieldHost, FieldProtocol,
		FieldReferer, FieldUserAgent, FieldStatus, FieldDurationNS,
		FieldDurationHuman, FieldRemoteAddr, FieldClientIP, FieldRequestID,
	}
	fieldMap := make(map[LogField]bool)
	for _, field := range cfg.Fields {
		fieldMap[field] = true
	}
	for _, expected := range expectedFields {
		if !fieldMap[expected] {
			t.Errorf("Expected field %s to be in default cfg", expected)
		}
	}
	if len(cfg.ExcludedPaths) != 0 {
		t.Errorf("Expected default excluded paths to be empty, got %d", len(cfg.ExcludedPaths))
	}
	if cfg.MaxBodySize != 1024 {
		t.Errorf("Expected default MaxBodySize to be 1024, got %d", cfg.MaxBodySize)
	}
	if len(cfg.SensitiveFields) == 0 {
		t.Error("Expected default SensitiveFields to be populated")
	}
}

func TestRequestLogger_RequestBodyLogging(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:         []LogField{FieldMethod, FieldRequestBody},
			LogRequestBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/test").
			WithBytes([]byte(`{"name":"test","value":123}`)).
			WithHeader(httpx.HeaderContentType, "application/json").
			Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); !found {
			t.Error("Expected request_body field to be present")
		} else if body, ok := value.(string); !ok || body != `{"name":"test","value":123}` {
			t.Errorf("Expected request_body to be '{\"name\":\"test\",\"value\":123}', got %v", value)
		}
	})

	t.Run("disabled by default", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger)(handler)

		req := zhtest.NewRequest(http.MethodPost, "/test").
			WithBytes([]byte(`{"name":"test"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		if _, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			t.Error("Expected request_body field not to be present when disabled")
		}
	})

	t.Run("body available to handler", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}

		// Handler that reads the body
		bodyReadingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_, _ = w.Write([]byte("read: " + string(body)))
		})

		middleware := New(logger, Config{
			Fields:         []LogField{FieldMethod, FieldRequestBody},
			LogRequestBody: true,
		})(bodyReadingHandler)

		req := zhtest.NewRequest(http.MethodPost, "/test").
			WithBytes([]byte(`{"data":"value"}`)).
			Build()
		recorder := zhtest.Serve(middleware, req)

		// Verify handler could read the body
		if !strings.Contains(recorder.Body.String(), `{"data":"value"}`) {
			t.Errorf("Expected handler to read body, got %s", recorder.Body.String())
		}
	})
}

func TestRequestLogger_ResponseBodyLogging(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok","id":123}`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldMethod, FieldResponseBody},
			LogResponseBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		if value, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); !found {
			t.Error("Expected response_body field to be present")
		} else if body, ok := value.(string); !ok {
			t.Errorf("Expected response_body to be string, got %T", value)
		} else {
			// Check for expected content without relying on key order
			if !strings.Contains(body, `"status":"ok"`) || !strings.Contains(body, `"id":123`) {
				t.Errorf("Expected response_body to contain '{\"status\":\"ok\",\"id\":123}', got %v", body)
			}
		}
	})

	t.Run("disabled by default", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		middleware := New(logger)(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		if _, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); found {
			t.Error("Expected response_body field not to be present when disabled")
		}
	})

	t.Run("response still returned", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`response data`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldStatus},
			LogResponseBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		recorder := zhtest.Serve(middleware, req)

		if recorder.Body.String() != "response data" {
			t.Errorf("Expected response body to be 'response data', got %s", recorder.Body.String())
		}
	})
}

func TestRequestLogger_MaxBodySize(t *testing.T) {
	t.Run("request body truncated", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:         []LogField{FieldRequestBody},
			LogRequestBody: true,
			MaxBodySize:    10,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/test").
			WithBytes([]byte(`this is a long request body`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			// 10 chars + "..." = 13
			if len(value.(string)) != 13 {
				t.Errorf("Expected request_body to be truncated to 10 chars + ..., got %d", len(value.(string)))
			}
			if !strings.HasSuffix(value.(string), "...") {
				t.Errorf("Expected request_body to end with ..., got %s", value.(string))
			}
		}
	})

	t.Run("response body truncated", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`this is a long response body`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldResponseBody},
			LogResponseBody: true,
			MaxBodySize:     10,
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); found {
			// 10 chars + "..." = 13
			if len(value.(string)) != 13 {
				t.Errorf("Expected response_body to be truncated to 10 chars + ..., got %d", len(value.(string)))
			}
			if !strings.HasSuffix(value.(string), "...") {
				t.Errorf("Expected response_body to end with ..., got %s", value.(string))
			}
		}
	})

	t.Run("unlimited body size", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`short`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldResponseBody},
			LogResponseBody: true,
			MaxBodySize:     -1, // Unlimited
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/test").Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); found {
			if value != "short" {
				t.Errorf("Expected response_body to be 'short', got %v", value)
			}
		}
	})
}

func TestRequestLogger_SensitiveFieldMasking(t *testing.T) {
	t.Run("masks password field", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:         []LogField{FieldRequestBody},
			LogRequestBody: true,
			MaxBodySize:    1024,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/login").
			WithBytes([]byte(`{"username":"admin","password":"secret123"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			body := value.(string)
			if strings.Contains(body, "secret123") {
				t.Errorf("Expected password to be masked, got %s", body)
			}
			if !strings.Contains(body, "[REDACTED]") {
				t.Errorf("Expected [REDACTED] in body, got %s", body)
			}
		}
	})

	t.Run("masks token field", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"access_token":"abc123","refresh_token":"xyz789"}`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldResponseBody},
			LogResponseBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/token").Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); found {
			body := value.(string)
			if strings.Contains(body, "abc123") || strings.Contains(body, "xyz789") {
				t.Errorf("Expected tokens to be masked, got %s", body)
			}
			count := strings.Count(body, "[REDACTED]")
			if count != 2 {
				t.Errorf("Expected 2 [REDACTED] values, got %d in %s", count, body)
			}
		}
	})

	t.Run("custom sensitive fields", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:          []LogField{FieldRequestBody},
			LogRequestBody:  true,
			SensitiveFields: []string{"ssn", "credit_card"},
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/payment").
			WithBytes([]byte(`{"name":"John","ssn":"123-45-6789","credit_card":"4111-1111-1111-1111"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			body := value.(string)
			if strings.Contains(body, "123-45-6789") || strings.Contains(body, "4111-1111-1111-1111") {
				t.Errorf("Expected sensitive fields to be masked, got %s", body)
			}
			// "name" should not be masked
			if !strings.Contains(body, "John") {
				t.Errorf("Expected name to not be masked, got %s", body)
			}
		}
	})

	t.Run("nested object masking", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:         []LogField{FieldRequestBody},
			LogRequestBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/api").
			WithBytes([]byte(`{"user":{"password":"nested_secret","name":"John"},"data":"value"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			body := value.(string)
			if strings.Contains(body, "nested_secret") {
				t.Errorf("Expected nested password to be masked, got %s", body)
			}
			// "name" inside user should not be masked
			if !strings.Contains(body, "John") {
				t.Errorf("Expected name to not be masked, got %s", body)
			}
		}
	})

	t.Run("array of objects masking", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:         []LogField{FieldRequestBody},
			LogRequestBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/api").
			WithBytes([]byte(`[{"id":1,"password":"pass1"},{"id":2,"password":"pass2"}]`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			body := value.(string)
			if strings.Contains(body, "pass1") || strings.Contains(body, "pass2") {
				t.Errorf("Expected passwords in array to be masked, got %s", body)
			}
			// Check ids are preserved
			if !strings.Contains(body, `"id":1`) || !strings.Contains(body, `"id":2`) {
				t.Errorf("Expected ids to not be masked, got %s", body)
			}
		}
	})

	t.Run("non-json body passthrough", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:         []LogField{FieldRequestBody},
			LogRequestBody: true,
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/api").
			WithBytes([]byte(`plain text body`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			if value != "plain text body" {
				t.Errorf("Expected plain text body, got %v", value)
			}
		}
	})

	t.Run("empty sensitive fields list", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:          []LogField{FieldRequestBody},
			LogRequestBody:  true,
			SensitiveFields: []string{}, // Empty but not nil
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/api").
			WithBytes([]byte(`{"password":"secret"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if value, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); found {
			body := value.(string)
			// With empty sensitive fields list, password should NOT be masked
			if !strings.Contains(body, "secret") {
				t.Errorf("Expected password not to be masked with empty list, got %s", body)
			}
		}
	})
}

func TestRequestLogger_BothBodies(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"echo":` + string(body) + `}`))
	})
	middleware := New(logger, Config{
		Fields:          []LogField{FieldRequestBody, FieldResponseBody},
		LogRequestBody:  true,
		LogResponseBody: true,
	})(handler)

	req := zhtest.NewRequest(http.MethodPost, "/echo").
		WithBytes([]byte(`{"msg":"hello"}`)).
		Build()
	recorder := zhtest.Serve(middleware, req)

	// Verify handler works
	if !strings.Contains(recorder.Body.String(), `"msg":"hello"`) {
		t.Errorf("Expected handler to work, got %s", recorder.Body.String())
	}

	// Verify both bodies logged
	if len(logger.infoLogs) != 1 {
		t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
	}

	if _, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); !found {
		t.Error("Expected request_body field to be present")
	}
	if _, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); !found {
		t.Error("Expected response_body field to be present")
	}
}

type panicLogger struct{}

func (p *panicLogger) Debug(msg string, fields ...log.Field) {}
func (p *panicLogger) Info(msg string, fields ...log.Field)  {}
func (p *panicLogger) Warn(msg string, fields ...log.Field)  {}
func (p *panicLogger) Error(msg string, fields ...log.Field) {}
func (p *panicLogger) Panic(msg string, fields ...log.Field) {
	panic(msg)
}
func (p *panicLogger) Fatal(msg string, fields ...log.Field)      {}
func (p *panicLogger) WithFields(fields ...log.Field) log.Logger  { return p }
func (p *panicLogger) WithContext(ctx context.Context) log.Logger { return p }

func TestRequestLogger_ExcludedPathsAndIncludedPathsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when both ExcludedPaths and IncludedPaths are set")
		} else if !strings.Contains(r.(string), "cannot set both ExcludedPaths and IncludedPaths") {
			t.Errorf("Expected panic message about ExcludedPaths and IncludedPaths, got: %v", r)
		}
	}()

	logger := &panicLogger{}
	_ = New(logger, Config{
		ExcludedPaths: []string{"/health"},
		IncludedPaths: []string{"/api/debug"},
	})
}

func TestRequestLogger_IncludedPaths(t *testing.T) {
	t.Run("body logging only for included paths", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldPath, FieldRequestBody, FieldResponseBody},
			LogRequestBody:  true,
			LogResponseBody: true,
			IncludedPaths:   []string{"/api/debug"},
		})(handler)

		// Request to allowed path - bodies should be logged
		req1 := zhtest.NewRequest(http.MethodPost, "/api/debug").
			WithBytes([]byte(`{"test":"data"}`)).
			Build()
		zhtest.Serve(middleware, req1)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}
		if _, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); !found {
			t.Error("Expected request_body to be present for allowed path")
		}
		if _, found := findFieldValue(logger.infoLogs[0].fields, "response_body"); !found {
			t.Error("Expected response_body to be present for allowed path")
		}

		// Request to non-allowed path - bodies should NOT be logged
		logger2 := &requestLoggerMockLogger{}
		middleware2 := New(logger2, Config{
			Fields:          []LogField{FieldPath, FieldRequestBody, FieldResponseBody},
			LogRequestBody:  true,
			LogResponseBody: true,
			IncludedPaths:   []string{"/api/debug"},
		})(handler)

		req2 := zhtest.NewRequest(http.MethodPost, "/api/other").
			WithBytes([]byte(`{"test":"data"}`)).
			Build()
		zhtest.Serve(middleware2, req2)

		if len(logger2.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger2.infoLogs))
		}
		if _, found := findFieldValue(logger2.infoLogs[0].fields, "request_body"); found {
			t.Error("Expected request_body to NOT be present for non-allowed path")
		}
		if _, found := findFieldValue(logger2.infoLogs[0].fields, "response_body"); found {
			t.Error("Expected response_body to NOT be present for non-allowed path")
		}
	})

	t.Run("prefix matching for included paths", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldPath, FieldRequestBody, FieldResponseBody},
			LogRequestBody:  true,
			LogResponseBody: true,
			IncludedPaths:   []string{"/api/debug/"},
		})(handler)

		// Request to path under allowed prefix
		req := zhtest.NewRequest(http.MethodPost, "/api/debug/test").
			WithBytes([]byte(`{"test":"data"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}
		if _, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); !found {
			t.Error("Expected request_body to be present for path under allowed prefix")
		}
	})

	t.Run("empty included paths allows all", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		middleware := New(logger, Config{
			Fields:          []LogField{FieldPath, FieldRequestBody, FieldResponseBody},
			LogRequestBody:  true,
			LogResponseBody: true,
			IncludedPaths:   []string{}, // Empty - should allow all
		})(handler)

		req := zhtest.NewRequest(http.MethodPost, "/api/anything").
			WithBytes([]byte(`{"test":"data"}`)).
			Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}
		if _, found := findFieldValue(logger.infoLogs[0].fields, "request_body"); !found {
			t.Error("Expected request_body to be present when IncludedPaths is empty")
		}
	})
}

type flusherRecorder struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (f *flusherRecorder) Flush() {
	f.flushed = true
}

func TestRequestLogger_Flush(t *testing.T) {
	tests := []struct {
		name              string
		underlyingFlusher bool
		expectFlushCalled bool
	}{
		{
			name:              "flush passes through to underlying Flusher",
			underlyingFlusher: true,
			expectFlushCalled: true,
		},
		{
			name:              "flush no-op when underlying doesn't implement Flusher",
			underlyingFlusher: false,
			expectFlushCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &requestLoggerMockLogger{}

			var base http.ResponseWriter
			var flushCalled *bool

			if tt.underlyingFlusher {
				rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}
				base = rec
				flushCalled = &rec.flushed
			} else {
				rec := httptest.NewRecorder()
				base = rec
				flushCalled = new(bool)
			}

			// Create bodyCapturingResponseWriter directly
			bcrw := &bodyCapturingResponseWriter{
				ResponseWriter: rwutil.NewResponseWriter(base),
			}

			// Call Flush
			bcrw.Flush()

			if *flushCalled != tt.expectFlushCalled {
				t.Errorf("expected flush called=%v, got=%v", tt.expectFlushCalled, *flushCalled)
			}

			_ = logger // suppress unused warning
		})
	}
}

func TestRequestLogger_Flush_SupportsSSE(t *testing.T) {
	logger := &requestLoggerMockLogger{}
	rec := &flusherRecorder{ResponseRecorder: httptest.NewRecorder()}

	middleware := New(logger, Config{
		Fields: []LogField{FieldPath, FieldStatus},
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get a Flusher from the writer
		f, ok := w.(http.Flusher)
		if !ok {
			t.Error("expected ResponseWriter to implement http.Flusher")
			return
		}

		// Write and flush like SSE would
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextEventStream)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
		f.Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	handler.ServeHTTP(rec, req)

	if !rec.flushed {
		t.Error("expected Flush to be called on underlying ResponseWriter")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequestLogger_CustomFieldsCallback(t *testing.T) {
	t.Run("custom fields from header", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields: []LogField{FieldMethod, FieldPath},
			CustomFields: func(r *http.Request) []log.Field {
				return []log.Field{
					log.F("api_key", r.Header.Get("X-API-Key")),
				}
			},
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/api/users").
			WithHeader("X-API-Key", "secret-key-123").
			Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		entry := logger.infoLogs[0]
		if value, found := findFieldValue(entry.fields, "api_key"); !found {
			t.Error("Expected api_key field to be present")
		} else if value != "secret-key-123" {
			t.Errorf("Expected api_key to be 'secret-key-123', got %v", value)
		}
	})

	type contextKey string

	const userIDKey contextKey = "user_id"
	t.Run("custom fields from context", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields: []LogField{FieldMethod, FieldPath},
			CustomFields: func(r *http.Request) []log.Field {
				if userID := r.Context().Value(userIDKey); userID != nil {
					return []log.Field{log.F("user_id", userID)}
				}
				return nil
			},
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
		req = req.WithContext(context.WithValue(req.Context(), userIDKey, "user-456"))
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		entry := logger.infoLogs[0]
		if value, found := findFieldValue(entry.fields, "user_id"); !found {
			t.Error("Expected user_id field to be present")
		} else if value != "user-456" {
			t.Errorf("Expected user_id to be 'user-456', got %v", value)
		}
	})

	t.Run("nil custom fields callback", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields:       []LogField{FieldMethod, FieldPath},
			CustomFields: nil,
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		// Should only have the built-in fields
		entry := logger.infoLogs[0]
		if _, found := findFieldValue(entry.fields, "method"); !found {
			t.Error("Expected method field to be present")
		}
		if _, found := findFieldValue(entry.fields, "path"); !found {
			t.Error("Expected path field to be present")
		}
	})

	t.Run("empty custom fields returned", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields: []LogField{FieldMethod, FieldPath},
			CustomFields: func(r *http.Request) []log.Field {
				return []log.Field{} // Empty slice
			},
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		// Should only have the built-in fields
		entry := logger.infoLogs[0]
		if _, found := findFieldValue(entry.fields, "method"); !found {
			t.Error("Expected method field to be present")
		}
	})

	t.Run("multiple custom fields", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields: []LogField{FieldMethod},
			CustomFields: func(r *http.Request) []log.Field {
				return []log.Field{
					log.F("tenant_id", "tenant-123"),
					log.F("request_source", "internal"),
					log.F("api_version", "v2"),
				}
			},
		})(handler)

		req := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
		zhtest.Serve(middleware, req)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		entry := logger.infoLogs[0]
		expectedFields := map[string]any{
			"tenant_id":      "tenant-123",
			"request_source": "internal",
			"api_version":    "v2",
		}

		for key, expectedValue := range expectedFields {
			if value, found := findFieldValue(entry.fields, key); !found {
				t.Errorf("Expected %s field to be present", key)
			} else if value != expectedValue {
				t.Errorf("Expected %s to be '%v', got %v", key, expectedValue, value)
			}
		}
	})

	t.Run("conditional custom fields", func(t *testing.T) {
		logger := &requestLoggerMockLogger{}
		handler := &statusTestHandler{statusCode: http.StatusOK}
		middleware := New(logger, Config{
			Fields: []LogField{FieldPath},
			CustomFields: func(r *http.Request) []log.Field {
				var fields []log.Field
				if strings.HasPrefix(r.URL.Path, "/admin/") {
					fields = append(fields, log.F("access_level", "admin"))
				}
				if r.Header.Get("X-Internal") == "true" {
					fields = append(fields, log.F("request_source", "internal"))
				}
				return fields
			},
		})(handler)

		// Request to admin path with internal header
		req1 := zhtest.NewRequest(http.MethodGet, "/admin/users").
			WithHeader("X-Internal", "true").
			Build()
		zhtest.Serve(middleware, req1)

		if len(logger.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger.infoLogs))
		}

		entry := logger.infoLogs[0]
		if _, found := findFieldValue(entry.fields, "access_level"); !found {
			t.Error("Expected access_level field to be present for admin path")
		}
		if _, found := findFieldValue(entry.fields, "request_source"); !found {
			t.Error("Expected request_source field to be present")
		}

		// Request to non-admin path without internal header
		logger2 := &requestLoggerMockLogger{}
		middleware2 := New(logger2, Config{
			Fields: []LogField{FieldPath},
			CustomFields: func(r *http.Request) []log.Field {
				var fields []log.Field
				if strings.HasPrefix(r.URL.Path, "/admin/") {
					fields = append(fields, log.F("access_level", "admin"))
				}
				if r.Header.Get("X-Internal") == "true" {
					fields = append(fields, log.F("request_source", "internal"))
				}
				return fields
			},
		})(handler)

		req2 := zhtest.NewRequest(http.MethodGet, "/api/users").Build()
		zhtest.Serve(middleware2, req2)

		if len(logger2.infoLogs) != 1 {
			t.Fatalf("Expected 1 info log, got %d", len(logger2.infoLogs))
		}

		entry2 := logger2.infoLogs[0]
		if _, found := findFieldValue(entry2.fields, "access_level"); found {
			t.Error("Expected access_level field NOT to be present for non-admin path")
		}
		if _, found := findFieldValue(entry2.fields, "request_source"); found {
			t.Error("Expected request_source field NOT to be present")
		}
	})
}

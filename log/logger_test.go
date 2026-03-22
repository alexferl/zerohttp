package log

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
)

func TestFieldHelpers(t *testing.T) {
	t.Run("F helper", func(t *testing.T) {
		f := F("key", "value")
		if f.Key != "key" {
			t.Errorf("expected key 'key', got '%s'", f.Key)
		}
		if f.Value != "value" {
			t.Errorf("expected value 'value', got '%v'", f.Value)
		}
	})

	t.Run("E helper", func(t *testing.T) {
		e := E("some error")
		if e.Key != "error" {
			t.Errorf("expected key 'error', got '%s'", e.Key)
		}
		if e.Value != "some error" {
			t.Errorf("expected value 'some error', got '%v'", e.Value)
		}
	})

	t.Run("P helper", func(t *testing.T) {
		p := P("panic msg")
		if p.Key != "panic" {
			t.Errorf("expected key 'panic', got '%s'", p.Key)
		}
		if p.Value != "panic msg" {
			t.Errorf("expected value 'panic msg', got '%v'", p.Value)
		}
	})
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Fatal("NewDefaultLogger returned nil")
	}
	if logger.logger == nil {
		t.Error("DefaultLogger.logger is nil")
	}
	if logger.fields == nil {
		t.Error("DefaultLogger.fields is nil")
	}
	if len(logger.fields) != 0 {
		t.Errorf("expected empty fields slice, got length %d", len(logger.fields))
	}
}

func createTestLogger() (*DefaultLogger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := &DefaultLogger{
		logger: log.New(buf, "", 0), // No timestamps for predictable output
		fields: make([]Field, 0),
	}
	return logger, buf
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*DefaultLogger, string, ...Field)
		expected string
	}{
		{"Debug", (*DefaultLogger).Debug, "[DBG]"},
		{"Info", (*DefaultLogger).Info, "[INF]"},
		{"Warn", (*DefaultLogger).Warn, "[WRN]"},
		{"Error", (*DefaultLogger).Error, "[ERR]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, buf := createTestLogger()
			tt.logFunc(logger, "test message")

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain '%s', got '%s'", tt.expected, output)
			}
			if !strings.Contains(output, "test message") {
				t.Errorf("expected output to contain 'test message', got '%s'", output)
			}
		})
	}
}

func TestLogWithFields(t *testing.T) {
	logger, buf := createTestLogger()
	logger.Info("test", F("key1", "value1"), F("key2", 42))

	output := buf.String()
	expected := []string{"[INF]", "test", "key1=value1", "key2=42"}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain '%s', got '%s'", exp, output)
		}
	}
}

func TestLogPanic(t *testing.T) {
	logger, buf := createTestLogger()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but none occurred")
		} else if r != "panic message" {
			t.Errorf("expected panic message 'panic message', got '%v'", r)
		}

		output := buf.String()
		if !strings.Contains(output, "[PNC]") {
			t.Errorf("expected output to contain '[PNC]', got '%s'", output)
		}
		if !strings.Contains(output, "panic message") {
			t.Errorf("expected output to contain 'panic message', got '%s'", output)
		}
	}()

	logger.Panic("panic message")
}

func TestWithFields(t *testing.T) {
	logger, buf := createTestLogger()

	loggerWithFields := logger.WithFields(F("base", "value"), F("count", 1))

	if loggerWithFields == logger {
		t.Error("WithFields should return a new logger instance")
	}

	loggerWithFields.Info("test message", F("extra", "field"))

	output := buf.String()
	expected := []string{"[INF]", "test message", "base=value", "count=1", "extra=field"}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain '%s', got '%s'", exp, output)
		}
	}
}

func TestWithContext(t *testing.T) {
	logger, _ := createTestLogger()
	ctx := context.Background()

	loggerWithCtx := logger.WithContext(ctx)

	// For DefaultLogger, should return the same instance
	if loggerWithCtx != logger {
		t.Error("WithContext should return the same logger instance for DefaultLogger")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"error", errors.New("test error"), "test error"},
		{"nil", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			if result != tt.expected {
				t.Errorf("formatValue(%v) = '%s', expected '%s'", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChainedWithFields(t *testing.T) {
	logger, buf := createTestLogger()

	logger1 := logger.WithFields(F("step", 1))
	logger2 := logger1.WithFields(F("step", 2), F("extra", "data"))

	logger2.Info("chained test")

	output := buf.String()
	expected := []string{"step=1", "step=2", "extra=data"}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain '%s', got '%s'", exp, output)
		}
	}
}

func TestStdLogger(t *testing.T) {
	logger, buf := createTestLogger()

	stdLogger := StdLogger(logger)

	stdLogger.Println("TLS handshake error: test")

	output := buf.String()
	if !strings.Contains(output, "[ERR]") {
		t.Errorf("expected output to contain '[ERR]', got '%s'", output)
	}
	if !strings.Contains(output, "TLS handshake error: test") {
		t.Errorf("expected output to contain 'TLS handshake error: test', got '%s'", output)
	}
	if strings.Contains(output, "\n\n") {
		t.Errorf("expected no double newlines, output had double newline: '%s'", output)
	}
}

func TestLogWriter(t *testing.T) {
	logger, buf := createTestLogger()
	writer := &logWriter{logger: logger}

	n, err := writer.Write([]byte("test message\n"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 13 {
		t.Errorf("expected 13 bytes written, got %d", n)
	}

	output := buf.String()
	if !strings.Contains(output, "[ERR]") {
		t.Errorf("expected output to contain '[ERR]', got '%s'", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got '%s'", output)
	}
	if strings.Contains(output, "\n\n") {
		t.Errorf("expected no double newlines from trimming, got: '%s'", output)
	}
}

func TestSetColorize(t *testing.T) {
	logger, buf := createTestLogger()

	// Test disabling colors
	logger.SetColorize(false)
	logger.Info("no color message")

	output := buf.String()
	if !strings.Contains(output, "[INF]") {
		t.Errorf("expected output to contain '[INF]', got '%s'", output)
	}
	if !strings.Contains(output, "no color message") {
		t.Errorf("expected output to contain 'no color message', got '%s'", output)
	}
	// With colors disabled, there should be no ANSI codes
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when colorize is disabled, got '%s'", output)
	}

	// Test re-enabling colors
	buf.Reset()
	logger.SetColorize(true)
	logger.Info("color message")

	output = buf.String()
	if !strings.Contains(output, "[INF]") {
		t.Errorf("expected output to contain '[INF]', got '%s'", output)
	}
}

func TestLogWithColorsEnabled(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(true)

	logger.Info("colored info")
	output := buf.String()

	// Should contain ANSI color codes
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected ANSI codes when colors enabled, got '%s'", output)
	}
}

func TestLogWithColorsDisabled(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(false)

	logger.Info("plain info")
	output := buf.String()

	// Should NOT contain ANSI color codes
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes when colors disabled, got '%s'", output)
	}
}

func TestShouldColorize_NO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result := shouldColorize()
	if result {
		t.Error("shouldColorize should return false when NO_COLOR is set")
	}
}

func TestShouldColorize_CI(t *testing.T) {
	tests := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS"}

	for _, envVar := range tests {
		t.Run(envVar, func(t *testing.T) {
			t.Setenv(envVar, "true")
			result := shouldColorize()
			if result {
				t.Errorf("shouldColorize should return false when %s is set", envVar)
			}
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()
	defer SetGlobalLogger(original)

	// Set up test logger
	testLogger := NewDefaultLogger()
	SetGlobalLogger(testLogger)

	// Test GetGlobalLogger returns our test logger
	if GetGlobalLogger() != testLogger {
		t.Error("GetGlobalLogger should return the set global logger")
	}
}

func TestGlobalLoggerMethods(t *testing.T) {
	// Save and restore original
	original := GetGlobalLogger()
	defer SetGlobalLogger(original)

	// Use NoopLogger to avoid output during tests
	SetGlobalLogger(&NoopLogger{})

	// Test that global methods don't panic via GetGlobalLogger
	GetGlobalLogger().Debug("debug message", F("key", "value"))
	GetGlobalLogger().Info("info message", F("key", "value"))
	GetGlobalLogger().Warn("warn message", F("key", "value"))
	GetGlobalLogger().Error("error message", F("key", "value"))

	// Test WithFields chaining on global
	logger := GetGlobalLogger().WithFields(F("field", "value"))
	logger.Info("chained message")
}

func TestLogFatal(t *testing.T) {
	logger, _ := createTestLogger()

	// We can't actually test Fatal since it calls os.Exit
	// Just verify the method exists and has correct signature by compiling
	_ = logger.Fatal
}

func TestLogWithAllLevels(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(false)

	tests := []struct {
		name     string
		logFunc  func(string, ...Field)
		expected string
	}{
		{"Debug", logger.Debug, "[DBG]"},
		{"Info", logger.Info, "[INF]"},
		{"Warn", logger.Warn, "[WRN]"},
		{"Error", logger.Error, "[ERR]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc("test")
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain '%s', got '%s'", tt.expected, output)
			}
		})
	}
}

func TestLogWithFieldsColorKeys(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(true)

	logger.Info("test", F("mykey", "myvalue"))
	output := buf.String()

	// Should contain the key with cyan color
	if !strings.Contains(output, "mykey") {
		t.Errorf("expected output to contain 'mykey', got '%s'", output)
	}
	if !strings.Contains(output, "myvalue") {
		t.Errorf("expected output to contain 'myvalue', got '%s'", output)
	}
}

func TestLogAllLevelColors(t *testing.T) {
	logger, _ := createTestLogger()
	logger.SetColorize(true)

	// Just ensure all levels work with colors
	levels := []struct {
		name string
		fn   func(string, ...Field)
	}{
		{"DBG", logger.Debug},
		{"INF", logger.Info},
		{"WRN", logger.Warn},
		{"ERR", logger.Error},
	}

	for _, level := range levels {
		level.fn("test message for " + level.name)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		setLevel      LogLevel
		logLevel      LogLevel
		shouldLog     bool
		expectedLevel string
	}{
		{"Debug at Debug level", DebugLevel, DebugLevel, true, "[DBG]"},
		{"Info at Debug level", DebugLevel, InfoLevel, true, "[INF]"},
		{"Error at Debug level", DebugLevel, ErrorLevel, true, "[ERR]"},
		{"Debug at Info level", InfoLevel, DebugLevel, false, ""},
		{"Info at Info level", InfoLevel, InfoLevel, true, "[INF]"},
		{"Warn at Info level", InfoLevel, WarnLevel, true, "[WRN]"},
		{"Debug at Warn level", WarnLevel, DebugLevel, false, ""},
		{"Info at Warn level", WarnLevel, InfoLevel, false, ""},
		{"Warn at Warn level", WarnLevel, WarnLevel, true, "[WRN]"},
		{"Error at Warn level", WarnLevel, ErrorLevel, true, "[ERR]"},
		{"Info at Error level", ErrorLevel, InfoLevel, false, ""},
		{"Error at Error level", ErrorLevel, ErrorLevel, true, "[ERR]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, buf := createTestLogger()
			logger.SetLevel(tt.setLevel)

			switch tt.logLevel {
			case DebugLevel:
				logger.Debug("test message")
			case InfoLevel:
				logger.Info("test message")
			case WarnLevel:
				logger.Warn("test message")
			case ErrorLevel:
				logger.Error("test message")
			}

			output := buf.String()
			if tt.shouldLog {
				if !strings.Contains(output, tt.expectedLevel) {
					t.Errorf("expected output to contain '%s', got '%s'", tt.expectedLevel, output)
				}
				if !strings.Contains(output, "test message") {
					t.Errorf("expected output to contain 'test message', got '%s'", output)
				}
			} else {
				if output != "" {
					t.Errorf("expected no output, got '%s'", output)
				}
			}
		})
	}
}

func TestDefaultLogLevel(t *testing.T) {
	logger := NewDefaultLogger()
	if logger.GetLevel() != InfoLevel {
		t.Errorf("expected default log level to be InfoLevel, got %v", logger.GetLevel())
	}
}

func TestSetAndGetLevel(t *testing.T) {
	logger := NewDefaultLogger()

	// Test setting different levels
	levels := []LogLevel{DebugLevel, InfoLevel, WarnLevel, ErrorLevel, PanicLevel, FatalLevel}
	for _, level := range levels {
		logger.SetLevel(level)
		if logger.GetLevel() != level {
			t.Errorf("expected level %v, got %v", level, logger.GetLevel())
		}
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DebugLevel, "DBG"},
		{InfoLevel, "INF"},
		{WarnLevel, "WRN"},
		{ErrorLevel, "ERR"},
		{PanicLevel, "PNC"},
		{FatalLevel, "FTL"},
		{LogLevel(999), "UNK"}, // Unknown level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestWithFieldsPreservesLevel(t *testing.T) {
	logger, _ := createTestLogger()
	logger.SetLevel(WarnLevel)

	newLogger := logger.WithFields(F("key", "value"))
	defaultLogger, ok := newLogger.(*DefaultLogger)
	if !ok {
		t.Fatal("WithFields should return *DefaultLogger")
	}

	if defaultLogger.GetLevel() != WarnLevel {
		t.Errorf("expected level to be preserved as WarnLevel, got %v", defaultLogger.GetLevel())
	}
}

func TestPanicWithLevelFiltering(t *testing.T) {
	// Test that Panic always panics even when filtered
	logger, buf := createTestLogger()
	logger.SetLevel(FatalLevel) // Set level higher than Panic

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic but none occurred")
		}
		// Panic should not have logged since level is Fatal
		output := buf.String()
		if strings.Contains(output, "[PNC]") {
			t.Error("Panic should not log when level is Fatal")
		}
	}()

	logger.Panic("panic message")
}

func TestFatalWithLevelFiltering(t *testing.T) {
	// We can't test Fatal since it calls os.Exit, but we can verify it exists
	logger, _ := createTestLogger()
	logger.SetLevel(FatalLevel)
	_ = logger.Fatal
}

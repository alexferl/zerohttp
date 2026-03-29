package log

import (
	"bytes"
	"context"
	"errors"
	"log"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestFieldHelpers(t *testing.T) {
	t.Run("F helper", func(t *testing.T) {
		f := F("key", "value")
		zhtest.AssertEqual(t, "key", f.Key)
		zhtest.AssertEqual(t, "value", f.Value)
	})

	t.Run("E helper", func(t *testing.T) {
		e := E("some error")
		zhtest.AssertEqual(t, "error", e.Key)
		zhtest.AssertEqual(t, "some error", e.Value)
	})

	t.Run("P helper", func(t *testing.T) {
		p := P("panic msg")
		zhtest.AssertEqual(t, "panic", p.Key)
		zhtest.AssertEqual(t, "panic msg", p.Value)
	})
}

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	zhtest.AssertNotNil(t, logger)
	zhtest.AssertNotNil(t, logger.logger)
	zhtest.AssertNotNil(t, logger.fields)
	zhtest.AssertEqual(t, 0, len(logger.fields))
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
			zhtest.AssertContains(t, output, tt.expected)
			zhtest.AssertContains(t, output, "test message")
		})
	}
}

func TestLogWithFields(t *testing.T) {
	logger, buf := createTestLogger()
	logger.Info("test", F("key1", "value1"), F("key2", 42))

	output := buf.String()
	expected := []string{"[INF]", "test", "key1=value1", "key2=42"}

	for _, exp := range expected {
		zhtest.AssertContains(t, output, exp)
	}
}

func TestLogPanic(t *testing.T) {
	logger, buf := createTestLogger()

	zhtest.AssertPanic(t, func() {
		logger.Panic("panic message")
	})

	output := buf.String()
	zhtest.AssertContains(t, output, "[PNC]")
	zhtest.AssertContains(t, output, "panic message")
}

func TestWithFields(t *testing.T) {
	logger, buf := createTestLogger()

	loggerWithFields := logger.WithFields(F("base", "value"), F("count", 1))

	zhtest.AssertTrue(t, loggerWithFields != logger)

	loggerWithFields.Info("test message", F("extra", "field"))

	output := buf.String()
	expected := []string{"[INF]", "test message", "base=value", "count=1", "extra=field"}

	for _, exp := range expected {
		zhtest.AssertContains(t, output, exp)
	}
}

func TestWithContext(t *testing.T) {
	logger, _ := createTestLogger()
	ctx := context.Background()

	loggerWithCtx := logger.WithContext(ctx)

	// For DefaultLogger, should return the same instance
	zhtest.AssertEqual(t, logger, loggerWithCtx)
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
			zhtest.AssertEqual(t, tt.expected, result)
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
		zhtest.AssertContains(t, output, exp)
	}
}

func TestStdLogger(t *testing.T) {
	logger, buf := createTestLogger()

	stdLogger := StdLogger(logger)

	stdLogger.Println("TLS handshake error: test")

	output := buf.String()
	zhtest.AssertContains(t, output, "[ERR]")
	zhtest.AssertContains(t, output, "TLS handshake error: test")
	zhtest.AssertNotContains(t, output, "\n\n")
}

func TestLogWriter(t *testing.T) {
	logger, buf := createTestLogger()
	writer := &logWriter{logger: logger}

	n, err := writer.Write([]byte("test message\n"))
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 13, n)

	output := buf.String()
	zhtest.AssertContains(t, output, "[ERR]")
	zhtest.AssertContains(t, output, "test message")
	zhtest.AssertNotContains(t, output, "\n\n")
}

func TestSetColorize(t *testing.T) {
	logger, buf := createTestLogger()

	// Test disabling colors
	logger.SetColorize(false)
	logger.Info("no color message")

	output := buf.String()
	zhtest.AssertContains(t, output, "[INF]")
	zhtest.AssertContains(t, output, "no color message")
	// With colors disabled, there should be no ANSI codes
	zhtest.AssertNotContains(t, output, "\033[")

	// Test re-enabling colors
	buf.Reset()
	logger.SetColorize(true)
	logger.Info("color message")

	output = buf.String()
	zhtest.AssertContains(t, output, "[INF]")
}

func TestLogWithColorsEnabled(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(true)

	logger.Info("colored info")
	output := buf.String()

	// Should contain ANSI color codes
	zhtest.AssertContains(t, output, "\033[")
}

func TestLogWithColorsDisabled(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(false)

	logger.Info("plain info")
	output := buf.String()

	// Should NOT contain ANSI color codes
	zhtest.AssertNotContains(t, output, "\033[")
}

func TestShouldColorize_NO_COLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result := shouldColorize()
	zhtest.AssertFalse(t, result)
}

func TestShouldColorize_CI(t *testing.T) {
	tests := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS"}

	for _, envVar := range tests {
		t.Run(envVar, func(t *testing.T) {
			t.Setenv(envVar, "true")
			result := shouldColorize()
			zhtest.AssertFalse(t, result)
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
	zhtest.AssertEqual(t, testLogger, GetGlobalLogger())
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
			zhtest.AssertContains(t, output, tt.expected)
		})
	}
}

func TestLogWithFieldsColorKeys(t *testing.T) {
	logger, buf := createTestLogger()
	logger.SetColorize(true)

	logger.Info("test", F("mykey", "myvalue"))
	output := buf.String()

	// Should contain the key with cyan color
	zhtest.AssertContains(t, output, "mykey")
	zhtest.AssertContains(t, output, "myvalue")
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
				zhtest.AssertContains(t, output, tt.expectedLevel)
				zhtest.AssertContains(t, output, "test message")
			} else {
				zhtest.AssertEmpty(t, output)
			}
		})
	}
}

func TestDefaultLogLevel(t *testing.T) {
	logger := NewDefaultLogger()
	zhtest.AssertEqual(t, InfoLevel, logger.GetLevel())
}

func TestSetAndGetLevel(t *testing.T) {
	logger := NewDefaultLogger()

	// Test setting different levels
	levels := []LogLevel{DebugLevel, InfoLevel, WarnLevel, ErrorLevel, PanicLevel, FatalLevel}
	for _, level := range levels {
		logger.SetLevel(level)
		zhtest.AssertEqual(t, level, logger.GetLevel())
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
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestWithFieldsPreservesLevel(t *testing.T) {
	logger, _ := createTestLogger()
	logger.SetLevel(WarnLevel)

	newLogger := logger.WithFields(F("key", "value"))
	defaultLogger, ok := newLogger.(*DefaultLogger)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, WarnLevel, defaultLogger.GetLevel())
}

func TestPanicWithLevelFiltering(t *testing.T) {
	// Test that Panic always panics even when filtered
	logger, buf := createTestLogger()
	logger.SetLevel(FatalLevel) // Set level higher than Panic

	zhtest.AssertPanic(t, func() {
		logger.Panic("panic message")
	})

	// Panic should not have logged since level is Fatal
	output := buf.String()
	zhtest.AssertNotContains(t, output, "[PNC]")
}

func TestFatalWithLevelFiltering(t *testing.T) {
	// We can't test Fatal since it calls os.Exit, but we can verify it exists
	logger, _ := createTestLogger()
	logger.SetLevel(FatalLevel)
	_ = logger.Fatal
}

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
		{"Debug", (*DefaultLogger).Debug, "[DEBUG]"},
		{"Info", (*DefaultLogger).Info, "[INFO]"},
		{"Warn", (*DefaultLogger).Warn, "[WARN]"},
		{"Error", (*DefaultLogger).Error, "[ERROR]"},
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
	expected := []string{"[INFO]", "test", "key1=value1", "key2=42"}

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
		if !strings.Contains(output, "[PANIC]") {
			t.Errorf("expected output to contain '[PANIC]', got '%s'", output)
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
	expected := []string{"[INFO]", "test message", "base=value", "count=1", "extra=field"}

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

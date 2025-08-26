package log

import (
	"context"
	"fmt"
	"log"
	"os"
)

// Logger defines the interface for logging in the framework
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...Field)
	// Info logs an info message
	Info(msg string, fields ...Field)
	// Warn logs a warning message
	Warn(msg string, fields ...Field)
	// Error logs an error message
	Error(msg string, fields ...Field)
	// Panic logs a panic message and panics
	Panic(msg string, fields ...Field)
	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...Field)

	// WithFields returns a logger with additional fields
	WithFields(fields ...Field) Logger
	// WithContext returns a logger with context
	WithContext(ctx context.Context) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value any
}

// F is a helper function to create a Field
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// E is a helper function to create a Field with the key set to "error"
func E(value any) Field {
	return Field{Key: "error", Value: value}
}

// P is a helper function to create a Field with the key set to "panic"
func P(value any) Field {
	return Field{Key: "panic", Value: value}
}

// DefaultLogger is a simple implementation using Go's standard log package
type DefaultLogger struct {
	logger *log.Logger
	fields []Field
}

// NewDefaultLogger creates a new default logger instance.
// It uses Go's standard log package with stdout output and standard flags.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		fields: make([]Field, 0),
	}
}

// Debug logs a debug message with optional fields
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	l.logWithLevel("DEBUG", msg, fields...)
}

// Info logs an info message with optional fields
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	l.logWithLevel("INFO", msg, fields...)
}

// Warn logs a warning message with optional fields
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	l.logWithLevel("WARN", msg, fields...)
}

// Error logs an error message with optional fields
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	l.logWithLevel("ERROR", msg, fields...)
}

// Panic logs a panic message with optional fields and then panics
func (l *DefaultLogger) Panic(msg string, fields ...Field) {
	l.logWithLevel("PANIC", msg, fields...)
	panic(msg)
}

// Fatal logs a fatal message with optional fields and then exits with code 1
func (l *DefaultLogger) Fatal(msg string, fields ...Field) {
	l.logWithLevel("FATAL", msg, fields...)
	os.Exit(1)
}

// WithFields creates a new logger instance with additional fields.
// The fields are combined with any existing fields on the logger.
func (l *DefaultLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &DefaultLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithContext creates a new logger instance with context.
// For the default logger implementation, context doesn't change behavior.
func (l *DefaultLogger) WithContext(ctx context.Context) Logger {
	// For the default logger, context doesn't change behavior
	return l
}

// logWithLevel logs a message at the specified level with fields.
// It formats the message with the level prefix and appends all fields.
func (l *DefaultLogger) logWithLevel(level, msg string, fields ...Field) {
	allFields := append(l.fields, fields...)

	logMsg := "[" + level + "] " + msg
	if len(allFields) > 0 {
		logMsg += " |"
		for _, field := range allFields {
			logMsg += " " + field.Key + "=" + formatValue(field.Value)
		}
	}

	l.logger.Println(logMsg)
}

// formatValue converts a field value to its string representation.
// It handles strings, errors, and other types using appropriate formatting.
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return fmt.Sprint(val)
	}
}

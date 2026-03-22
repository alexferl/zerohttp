package log

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

// ANSI color codes for terminal output
const (
	colorReset       = "\033[0m"
	colorRed         = "\033[31m"
	colorGreen       = "\033[32m"
	colorYellow      = "\033[33m"
	colorBlue        = "\033[34m"
	colorMagenta     = "\033[35m"
	colorCyan        = "\033[36m"
	colorGray        = "\033[90m"
	colorWhiteBold   = "\033[1;37m"
	colorWhiteOnRed  = "\033[97;41m"
	colorWhiteOnBlue = "\033[97;44m"
)

// levelColors maps log levels to ANSI colors
var levelColors = map[string]string{
	"DBG": colorWhiteOnBlue,
	"INF": colorGreen,
	"WRN": colorYellow,
	"ERR": colorRed,
	"PNC": colorMagenta,
	"FTL": colorWhiteOnRed,
}

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

// Ensure DefaultLogger implements Logger
var _ Logger = (*DefaultLogger)(nil)

// DefaultLogger is a simple implementation using Go's standard log package
type DefaultLogger struct {
	logger   *log.Logger
	fields   []Field
	colorize bool
}

// NewDefaultLogger creates a new default logger instance.
// It uses Go's standard log package with stdout output and standard flags.
// Colors are enabled by default for TTY terminals, unless NO_COLOR is set
// or running in a CI environment.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger:   log.New(os.Stdout, "", log.LstdFlags),
		fields:   make([]Field, 0),
		colorize: shouldColorize(),
	}
}

// shouldColorize returns true if colors should be enabled.
// It checks for NO_COLOR environment variable and common CI environments.
func shouldColorize() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for common CI environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "BUILD_ID", "BUILD_NUMBER"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return false
		}
	}

	// Check for specific CI services
	ciServices := []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"JENKINS_URL",
		"BUILDKITE",
		"DRONE",
		"TEAMCITY_VERSION",
	}
	for _, v := range ciServices {
		if os.Getenv(v) != "" {
			return false
		}
	}

	return true
}

// SetColorize enables or disables colored output.
// When disabled, logs are output without ANSI color codes.
func (l *DefaultLogger) SetColorize(enabled bool) {
	l.colorize = enabled
}

// Debug logs a debug message with optional fields
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	l.logWithLevel("DBG", msg, fields...)
}

// Info logs an info message with optional fields
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	l.logWithLevel("INF", msg, fields...)
}

// Warn logs a warning message with optional fields
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	l.logWithLevel("WRN", msg, fields...)
}

// Error logs an error message with optional fields
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	l.logWithLevel("ERR", msg, fields...)
}

// Panic logs a panic message with optional fields and then panics
func (l *DefaultLogger) Panic(msg string, fields ...Field) {
	l.logWithLevel("PNC", msg, fields...)
	panic(msg)
}

// Fatal logs a fatal message with optional fields and then exits with code 1
func (l *DefaultLogger) Fatal(msg string, fields ...Field) {
	l.logWithLevel("FTL", msg, fields...)
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
// If colorize is enabled, it applies ANSI color codes to the level and field keys.
func (l *DefaultLogger) logWithLevel(level, msg string, fields ...Field) {
	allFields := append(l.fields, fields...)

	var b strings.Builder
	if l.colorize {
		levelColor := levelColors[level]
		if levelColor == "" {
			levelColor = colorReset
		}
		b.WriteString(levelColor)
		b.WriteString("[")
		b.WriteString(level)
		b.WriteString("]")
		b.WriteString(colorReset)
		b.WriteString(" ")
		b.WriteString(colorWhiteBold)
		b.WriteString(msg)
		b.WriteString(colorReset)
		if len(allFields) > 0 {
			b.WriteString(" |")
			for _, field := range allFields {
				b.WriteString(" ")
				b.WriteString(colorCyan)
				b.WriteString(field.Key)
				b.WriteString(colorReset)
				b.WriteString("=")
				b.WriteString(formatValue(field.Value))
			}
		}
	} else {
		b.WriteString("[")
		b.WriteString(level)
		b.WriteString("] ")
		b.WriteString(msg)
		if len(allFields) > 0 {
			b.WriteString(" |")
			for _, field := range allFields {
				b.WriteString(" ")
				b.WriteString(field.Key)
				b.WriteString("=")
				b.WriteString(formatValue(field.Value))
			}
		}
	}

	l.logger.Println(b.String())
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

// logWriter is an io.Writer adapter that writes to a Logger.
// It's used to bridge Go's standard log package with structured loggers.
type logWriter struct {
	logger Logger
}

// Write implements io.Writer, writing the message to the underlying logger.
// It trims trailing newlines for cleaner log output.
func (w *logWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSuffix(string(p), "\n")
	msg = strings.TrimSuffix(msg, "\r")
	w.logger.Error(msg)
	return len(p), nil
}

// StdLogger returns a standard library *log.Logger that writes to the provided Logger.
// This is useful for integrating with libraries that expect a standard logger,
// such as http.Server.ErrorLog.
//
// Example usage with http.Server:
//
//	logger := log.NewZerologLogger() // or any Logger implementation
//	server := &http.Server{
//	    ErrorLog: log.StdLogger(logger),
//	}
func StdLogger(logger Logger) *log.Logger {
	return log.New(&logWriter{logger: logger}, "", 0)
}

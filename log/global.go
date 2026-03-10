package log

import (
	"context"
	"sync"
)

// globalLogger is the package-level logger used when a logger cannot be passed.
// It defaults to a no-op logger to avoid nil pointer issues.
var (
	globalLogger   Logger = &NoopLogger{}
	globalLoggerMu sync.RWMutex
)

// SetGlobalLogger sets the package-level logger.
// This should be called during application initialization.
func SetGlobalLogger(l Logger) {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	globalLogger = l
}

// GetGlobalLogger returns the current package-level logger.
// Returns a no-op logger if none has been set.
func GetGlobalLogger() Logger {
	globalLoggerMu.RLock()
	defer globalLoggerMu.RUnlock()
	return globalLogger
}

// NoopLogger is a no-op implementation of Logger.
// Used as a safe default when no logger is configured.
type NoopLogger struct{}

func (n *NoopLogger) Debug(string, ...Field)             {}
func (n *NoopLogger) Info(string, ...Field)              {}
func (n *NoopLogger) Warn(string, ...Field)              {}
func (n *NoopLogger) Error(string, ...Field)             {}
func (n *NoopLogger) Panic(msg string, fields ...Field)  { panic(msg) }
func (n *NoopLogger) Fatal(string, ...Field)             {}
func (n *NoopLogger) WithFields(...Field) Logger         { return n }
func (n *NoopLogger) WithContext(context.Context) Logger { return n }

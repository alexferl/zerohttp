package log

import (
	"context"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestNoopLoggerMethods(t *testing.T) {
	logger := &NoopLogger{}

	// All these should execute without panic
	logger.Debug("debug message", F("key", "value"))
	logger.Info("info message", F("key", "value"))
	logger.Warn("warn message", F("key", "value"))
	logger.Error("error message", F("key", "value"))

	// Test WithFields returns a logger
	newLogger := logger.WithFields(F("newKey", "newValue"))
	zhtest.AssertNotNil(t, newLogger)

	// Test WithContext returns a logger
	ctxLogger := logger.WithContext(context.Background())
	zhtest.AssertNotNil(t, ctxLogger)

	// Test that chained calls work
	chainedLogger := logger.WithFields(F("key", "value")).WithContext(context.Background())
	zhtest.AssertNotNil(t, chainedLogger)
	chainedLogger.Info("test message")
}

func TestNoopLoggerPanic(t *testing.T) {
	logger := &NoopLogger{}

	zhtest.AssertPanic(t, func() {
		logger.Panic("panic message", F("key", "value"))
	})
}

func TestGlobalLoggerDefaultsToNoop(t *testing.T) {
	// Reset to default state
	SetGlobalLogger(&NoopLogger{})

	logger := GetGlobalLogger()
	zhtest.AssertNotNil(t, logger)

	// Should be able to call methods without panic
	logger.Info("test message")
}

func TestSetAndGetGlobalLogger(t *testing.T) {
	// Save original logger to restore later
	originalLogger := GetGlobalLogger()
	defer SetGlobalLogger(originalLogger)

	// Set a new logger
	newLogger := NewDefaultLogger()
	SetGlobalLogger(newLogger)

	// Retrieve it
	retrievedLogger := GetGlobalLogger()
	zhtest.AssertEqual(t, newLogger, retrievedLogger)
}

func TestGlobalLoggerThreadSafety(t *testing.T) {
	// Save original logger to restore later
	originalLogger := GetGlobalLogger()
	defer SetGlobalLogger(originalLogger)

	// Run concurrent operations
	done := make(chan bool)

	// Multiple goroutines setting the logger
	for range 10 {
		go func() {
			SetGlobalLogger(NewDefaultLogger())
			done <- true
		}()
	}

	// Multiple goroutines getting and using the logger
	for range 10 {
		go func() {
			logger := GetGlobalLogger()
			logger.Info("concurrent test message")
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 20 {
		<-done
	}
}

func TestNoopLoggerWithFieldsChaining(t *testing.T) {
	logger := &NoopLogger{}

	// WithFields should return the same instance for NoopLogger
	// (since there's nothing to track)
	newLogger := logger.WithFields(F("key", "value"))

	// It should not panic
	newLogger.Info("test")
}

type testContextKey string

func TestNoopLoggerWithContextChaining(t *testing.T) {
	logger := &NoopLogger{}
	ctx := context.WithValue(context.Background(), testContextKey("testKey"), "testValue")

	newLogger := logger.WithContext(ctx)

	// It should not panic
	newLogger.Info("test")
}

// Package log provides structured logging interfaces for zerohttp.
//
// This package defines a minimal logging interface that can be implemented
// by any logging library. A simple default implementation is provided,
// but users can bring their own logger (zap, zerolog, etc.).
//
// # Quick Start
//
// By default, zerohttp uses a simple built-in logger:
//
//	app := zh.New() // Uses default logger
//
// # Custom Logger
//
// Provide your own logger implementation:
//
//	app := zh.New(config.Config{
//	    Logger: myLogger,
//	})
//
// # Global Logger
//
// Access or set the global logger:
//
//	// Set global logger
//	log.SetGlobalLogger(myLogger)
//
//	// Get global logger
//	logger := log.GetGlobalLogger()
//	logger.Info("Server starting", log.F("port", 8080))
//
// # Log Levels
//
// The default logger supports log level filtering. Messages below the configured
// level are silently discarded.
//
// Available levels (in order of verbosity):
//
//	log.DebugLevel // 0 - most verbose
//	log.InfoLevel  // 1 - default
//	log.WarnLevel  // 2
//	log.ErrorLevel // 3
//	log.PanicLevel // 4
//	log.FatalLevel // 5 - least verbose
//
// Set the log level:
//
//	logger := log.NewDefaultLogger()
//	logger.SetLevel(log.DebugLevel) // Show all messages
//
// Get the current log level:
//
//	level := logger.GetLevel()
//
// # Using Fields
//
// Create structured log entries with fields:
//
//	logger.Info("Request completed",
//	    log.F("method", r.Method),
//	    log.F("path", r.URL.Path),
//	    log.F("status", 200),
//	    log.F("duration", time.Since(start)),
//	)
//
// # Error Logging
//
// Use the E helper for errors:
//
//	if err != nil {
//	    logger.Error("Database query failed", log.E(err))
//	}
//
// # Creating a Custom Logger
//
// Implement the [Logger] interface:
//
//	type MyLogger struct{}
//
//	func (l *MyLogger) Debug(msg string, fields ...log.Field) { }
//	func (l *MyLogger) Info(msg string, fields ...log.Field)  { }
//	func (l *MyLogger) Warn(msg string, fields ...log.Field)  { }
//	func (l *MyLogger) Error(msg string, fields ...log.Field) { }
//	func (l *MyLogger) Panic(msg string, fields ...log.Field) { }
//	func (l *MyLogger) Fatal(msg string, fields ...log.Field) { }
//	func (l *MyLogger) WithFields(fields ...log.Field) log.Logger { return l }
//	func (l *MyLogger) WithContext(ctx context.Context) log.Logger { return l }
//
// See examples/third_party/zerolog for a complete working example.
package log

// Package requestlogger provides HTTP request/response logging middleware.
//
// Logs incoming requests with method, path, status code, duration,
// and other details. Structured logging with configurable fields.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/requestlogger"
//
//	// Use defaults
//	app.Use(requestlogger.New())
//
//	// Custom configuration
//	app.Use(requestlogger.New(requestlogger.Config{
//	    SkipPaths: []string{"/health", "/metrics"},
//	    Fields: []string{"method", "path", "status", "duration", "ip"},
//	}))
//
// # Log Format
//
// Default format includes:
//   - timestamp
//   - method
//   - path
//   - status_code
//   - duration
//   - client_ip
//   - user_agent
package requestlogger

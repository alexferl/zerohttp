// Package recover provides panic recovery middleware.
//
// Catches panics from handlers and converts them to HTTP 500 errors
// with optional stack trace logging.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/recover"
//
//	// Use defaults
//	app.Use(recover.New())
//
//	// Custom error handler
//	app.Use(recover.New(recover.Config{
//	    ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
//	        log.Printf("Panic: %v\n%s", err, debug.Stack())
//	        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
//	    },
//	}))
//
// # Disable Stack Traces
//
//	app.Use(recover.New(recover.Config{
//	    LogStack: false,
//	}))
package recover

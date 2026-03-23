// Package value provides context value injection middleware.
//
// Injects static values into the request context for use by handlers.
// Useful for request-scoped configuration or dependencies.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/value"
//
//	// Inject static values
//	app.Use(value.New(value.Config{
//	    Values: map[string]any{
//	        "version":    "1.0.0",
//	        "request_id": uuid.New(),
//	    },
//	}))
//
// # Accessing Values
//
// Retrieve values in handlers:
//
//	version := r.Context().Value("version")
package value

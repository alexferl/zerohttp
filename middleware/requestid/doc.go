// Package requestid provides request ID generation middleware.
//
// Generates a unique request ID for each request, useful for tracing
// and logging. The ID is added to the request context and response headers.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/requestid"
//
//	// Use defaults (generates UUID)
//	app.Use(requestid.New())
//
//	// Custom configuration
//	app.Use(requestid.New(requestid.Config{
//	    HeaderName:     "X-Request-ID",
//	    Generator:      func() string { return myCustomID() },
//	    AllowFromHeader: true, // Accept client-provided IDs
//	}))
//
// # Accessing the Request ID
//
// Retrieve the ID in handlers:
//
//	id := requestid.Get(r)
//	log.Printf("Request ID: %s", id)
package requestid

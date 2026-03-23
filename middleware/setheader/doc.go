// Package setheader provides custom response header middleware.
//
// Sets static response headers on all responses. Useful for adding
// custom headers like X-Powered-By or custom security headers.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/setheader"
//
//	// Set custom headers
//	app.Use(setheader.New(setheader.Config{
//	    Headers: map[string]string{
//	        "X-Powered-By": "MyApp",
//	        "X-Version":    "1.0.0",
//	    },
//	}))
//
// # Conditional Headers
//
//	app.Use(setheader.New(setheader.Config{
//	    Headers: map[string]string{
//	        "X-Frame-Options": "DENY",
//	    },
//	    ExcludedPaths: []string{"/embed/*"},
//	}))
package setheader

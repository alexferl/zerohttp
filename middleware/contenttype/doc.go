// Package contenttype provides Content-Type validation middleware.
//
// Ensures requests have an acceptable Content-Type header.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/contenttype"
//
//	// Accept only JSON
//	app.Use(contenttype.New(contenttype.Config{
//	    AllowedTypes: []string{"application/json"},
//	}))
//
//	// Multiple types with wildcards
//	app.Use(contenttype.New(contenttype.Config{
//	    AllowedTypes: []string{"application/*", "text/plain"},
//	}))
package contenttype

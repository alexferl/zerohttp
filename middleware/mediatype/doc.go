// Package mediatype provides media type negotiation and validation middleware.
//
// Validates Accept headers against allowed media types and ensures
// clients can receive the response formats the server provides.
// Supports suffix matching (+json, +xml) and wildcard patterns.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/mediatype"
//
//	// Accept only specific vendor types
//	app.Use(mediatype.New(mediatype.Config{
//	    AllowedTypes: []string{"application/vnd.api+json"},
//	}))
//
//	// Accept any +json suffix
//	app.Use(mediatype.New(mediatype.Config{
//	    AllowedTypes: []string{"application/*+json"},
//	}))
//
//	// Accept multiple patterns
//	app.Use(mediatype.New(mediatype.Config{
//	    AllowedTypes: []string{
//	        "application/vnd.api+json",
//	        "application/vnd.company.*+json",
//	    },
//	}))
package mediatype

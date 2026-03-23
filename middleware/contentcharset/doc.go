// Package contentcharset provides Content-Type charset validation middleware.
//
// Ensures requests have an acceptable charset encoding.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/contentcharset"
//
//	// Accept only UTF-8
//	app.Use(contentcharset.New(contentcharset.Config{
//	    AllowedCharsets: []string{"utf-8"},
//	}))
//
//	// Multiple charsets
//	app.Use(contentcharset.New(contentcharset.Config{
//	    AllowedCharsets: []string{"utf-8", "iso-8859-1"},
//	}))
package contentcharset

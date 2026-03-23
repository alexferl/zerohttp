// Package contentencoding provides Content-Encoding validation middleware.
//
// Validates that the request Content-Encoding header is acceptable.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/contentencoding"
//
//	// Accept specific encodings
//	app.Use(contentencoding.New(contentencoding.Config{
//	    AllowedEncodings: []string{"gzip", "deflate", "identity"},
//	}))
package contentencoding

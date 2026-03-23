// Package nocache provides cache-busting middleware.
//
// Sets Cache-Control headers to prevent caching of dynamic responses.
// Useful for APIs, authenticated endpoints, and frequently changing content.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/nocache"
//
//	// Apply to all routes
//	app.Use(nocache.New())
//
//	// Apply to specific routes
//	app.Use(nocache.New(nocache.Config{
//	    IncludedPaths: []string{"/api/*"},
//	}))
//
// Sets headers:
//   - Cache-Control: no-store, no-cache, must-revalidate, proxy-revalidate
//   - Pragma: no-cache
//   - Expires: 0
package nocache

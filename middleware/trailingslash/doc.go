// Package trailingslash provides trailing slash normalization middleware.
//
// Redirects requests with or without trailing slashes to a canonical form.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/trailingslash"
//
//	// Remove trailing slashes (default)
//	app.Use(trailingslash.New())
//
//	// Add trailing slashes
//	app.Use(trailingslash.New(trailingslash.Config{
//	    Mode: trailingslash.Add,
//	}))
//
// # Skip Specific Paths
//
//	app.Use(trailingslash.New(trailingslash.Config{
//	    Mode:          trailingslash.Remove,
//	    ExcludedPaths: []string{"/api/*"},
//	}))
//
// # Modes
//
//   - Remove: Redirect /path/ to /path (default)
//   - Add: Redirect /path to /path/
//   - Strict: Return 404 for non-canonical paths
package trailingslash

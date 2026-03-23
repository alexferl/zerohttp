// Package etag provides ETag generation and validation middleware.
//
// ETags enable HTTP conditional requests, allowing clients to cache
// responses and avoid re-downloading unchanged content.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/etag"
//
//	// Use defaults (MD5-based ETags)
//	app.Use(etag.New())
//
//	// Custom configuration
//	app.Use(etag.New(etag.Config{
//	    Weak: true, // Use weak ETags (prefixed with W/)
//	}))
//
// The middleware automatically handles If-None-Match and If-Match headers,
// returning 304 Not Modified when content hasn't changed.
package etag

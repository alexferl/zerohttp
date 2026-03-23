// Package cache provides HTTP caching middleware with ETag and conditional request support.
//
// The cache middleware automatically generates ETags, handles conditional requests
// (If-None-Match, If-Modified-Since), and manages Cache-Control headers.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/cache"
//
//	// Use defaults (in-memory store)
//	app.Use(cache.New())
//
//	// Custom configuration
//	app.Use(cache.New(cache.Config{
//	    MaxEntries: 10000,
//	    TTL:        5 * time.Minute,
//	    StatusCodes: []int{200, 201, 404},
//	}))
//
// # Custom Store
//
// Implement the Store interface for Redis or other backends:
//
//	app.Use(cache.New(cache.Config{
//	    Store: myRedisStore,
//	}))
package cache

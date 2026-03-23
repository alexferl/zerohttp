// Package idempotency provides idempotent request handling middleware.
//
// Prevents duplicate processing of requests by tracking idempotency keys
// and returning cached responses for duplicate requests.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/idempotency"
//
//	// Use defaults (in-memory store)
//	app.Use(idempotency.New())
//
//	// Custom configuration
//	app.Use(idempotency.New(idempotency.Config{
//	    HeaderName: "Idempotency-Key",
//	    TTL:        24 * time.Hour,
//	}))
//
// # Custom Store
//
// Use Redis or other backends for distributed systems:
//
//	app.Use(idempotency.New(idempotency.Config{
//	    Store: myRedisIdempotencyStore,
//	}))
package idempotency

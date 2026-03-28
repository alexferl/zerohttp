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
//
// # Storage Adapter
//
// Use [NewStorageAdapter] to wrap a [storage.Storage] implementation:
//
//	import (
//	    "github.com/alexferl/zerohttp/middleware/idempotency"
//	    "github.com/alexferl/zerohttp/storage"
//	)
//
//	myStore := redis.New("localhost:6379") // implements storage.Storage and storage.Locker
//	adapter, err := idempotency.NewStorageAdapter(myStore)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	app.Use(idempotency.New(idempotency.Config{
//	    Store: adapter,
//	}))
//
// With custom lock TTL for long-running handlers:
//
//	adapter, err := idempotency.NewStorageAdapter(myStore, idempotency.StorageAdapterConfig{
//	    LockTTL: 5 * time.Minute,
//	})
package idempotency

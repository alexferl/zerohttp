// Package jwtauth provides JWT authentication middleware.
//
// The middleware provides pluggable JWT authentication. Users bring their own
// JWT library by implementing the Store interface.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/jwtauth"
//
//	app.Use(jwtauth.New(jwtauth.Config{
//	    Store: myStore,
//	    RequiredClaims: []string{"sub"},
//	}))
//
// # Built-in HS256
//
// For a zero-dependency option, use the built-in HS256 implementation:
//
//	app.Use(jwtauth.New(jwtauth.Config{
//	    Store: jwtauth.NewHS256Store(secret, opts),
//	}))
//
// # Token Revocation
//
// For token revocation (logout/refresh), implement Revoke and IsRevoked on your Store,
// or use the StorageAdapter with a shared storage backend:
//
//	// Share storage between idempotency, cache, and jwtauth
//	redisStorage := storage.NewRedisStorage(redisClient, storage.RedisStorageConfig{})
//	revocationStore := jwtauth.NewStorageAdapter(redisStorage)
//
//	// Use with your custom Store
//	myStore := &myStoreImpl{RevocationStore: revocationStore}
//
// # Accessing Claims
//
// Retrieve validated claims in handlers:
//
//	claims := jwtauth.GetClaims(r)
//	sub := claims.Subject()
//
// Security Note: The built-in HS256 uses HMAC-SHA256 symmetric signing.
// For asymmetric keys (RS256, ES256, EdDSA), use golang-jwt/jwt or lestrrat-go/jwx.
package jwtauth

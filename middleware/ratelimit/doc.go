// Package ratelimit provides rate limiting middleware.
//
// Supports token bucket and sliding window algorithms with configurable
// key extractors for per-client, per-user, or custom rate limiting.
//
// # Usage
//
//	import "github.com/alexferl/zerohttp/middleware/ratelimit"
//
//	// Per-IP rate limiting (default)
//	app.Use(ratelimit.New(ratelimit.Config{
//	    Rate:   100,             // 100 requests
//	    Window: time.Minute,     // per minute
//	}))
//
// # Key Extractors
//
// Rate limit by different criteria:
//
//	// By API key header
//	app.Use(ratelimit.New(ratelimit.Config{
//	    KeyExtractor: ratelimit.HeaderKeyExtractor("X-API-Key"),
//	}))
//
//	// By JWT subject (user), fallback to IP
//	app.Use(ratelimit.New(ratelimit.Config{
//	    KeyExtractor: ratelimit.CompositeKeyExtractor(
//	        ratelimit.JWTSubjectKeyExtractor(),
//	        ratelimit.IPKeyExtractor(),
//	    ),
//	}))
//
//	// By context value (e.g., user ID from auth middleware)
//	app.Use(ratelimit.New(ratelimit.Config{
//	    KeyExtractor: ratelimit.ContextKeyExtractor("user_id"),
//	}))
//
// # Custom Store
//
// Use Redis for distributed rate limiting:
//
//	app.Use(ratelimit.New(ratelimit.Config{
//	    Store: myRedisRateLimitStore,
//	}))
package ratelimit

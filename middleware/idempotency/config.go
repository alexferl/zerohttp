package idempotency

import (
	"context"
	"time"
)

// Store is the interface for idempotency storage backends.
// Users can implement this interface to provide their own storage
// (e.g., Redis, database, or distributed cache).
// Implement this if you only need idempotency storage.
// For a backend shared with other middlewares, implement storage.Storage
// and storage.Locker, then use idempotency.NewStorageAdapter instead.
type Store interface {
	// Get retrieves a cached response by key.
	// Returns the cached record, true if found, and any error.
	// If not found, returns false and nil error.
	// If an error occurs (e.g., network error), returns false and the error.
	Get(ctx context.Context, key string) (Record, bool, error)

	// Set stores a response in the cache with the given TTL.
	// Returns an error if the operation fails (e.g., network error for external stores).
	Set(ctx context.Context, key string, record Record, ttl time.Duration) error

	// Lock acquires an exclusive lock for the given key.
	// Returns true if the lock was acquired, false if the key is already locked (in-flight).
	// The lock should be released by calling Unlock.
	// Returns an error if the lock operation fails (e.g., network error).
	Lock(ctx context.Context, key string) (bool, error)

	// Unlock releases the lock for the given key.
	// Returns an error if the unlock operation fails.
	Unlock(ctx context.Context, key string) error

	// Close releases resources associated with the store.
	// Returns an error if the close operation fails.
	Close() error
}

// Record represents a cached idempotent response.
type Record struct {
	StatusCode int
	// Headers are stored as a flat slice [key1, val1, key2, val2, ...]
	// for efficient replay without map allocations.
	Headers   []string
	Body      []byte
	CreatedAt time.Time
}

// Config configures the idempotency middleware.
type Config struct {
	// HeaderName is the header to read the idempotency key from.
	// Default: "Idempotency-Key"
	HeaderName string

	// TTL is the cache duration for idempotent responses.
	// Default: 24h
	TTL time.Duration

	// MaxBodySize is the maximum request body size to hash (in bytes).
	// Requests with larger bodies are not cached.
	// Default: 1MB
	MaxBodySize int64

	// Store is a custom idempotency store implementation.
	// If nil, an in-memory store is used.
	// Default: nil
	Store Store

	// Required makes the idempotency key required on state-changing methods.
	// If true and no key is provided, returns 400 Bad Request.
	// Default: false
	Required bool

	// ExcludedPaths are paths that should skip idempotency check.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where idempotency check is explicitly applied.
	// If set, idempotency check will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, idempotency check applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// MaxKeys limits the number of unique keys stored in the default
	// in-memory store. Set to 0 for unlimited (not recommended).
	// Default: 10000
	MaxKeys int

	// LockRetryInterval is the initial interval between retries when waiting for
	// an in-flight request to complete. Uses exponential backoff with jitter.
	// Default: 10ms
	LockRetryInterval time.Duration

	// LockMaxRetries is the maximum number of retries when waiting for an
	// in-flight request to complete. After this is exhausted, a 409 Conflict
	// is returned.
	// Default: 300 (3 seconds at 10ms intervals)
	LockMaxRetries int

	// LockMaxInterval is the maximum interval between retries (caps exponential
	// backoff).
	// Default: 500ms
	LockMaxInterval time.Duration

	// LockBackoffMultiplier is the multiplier for exponential backoff.
	// Default: 2.0
	LockBackoffMultiplier float64
}

// DefaultConfig is the default configuration for the idempotency middleware.
var DefaultConfig = Config{
	HeaderName:            "Idempotency-Key",
	TTL:                   24 * time.Hour,
	MaxBodySize:           1024 * 1024,
	Required:              false,
	ExcludedPaths:         []string{},
	IncludedPaths:         []string{},
	MaxKeys:               10000,
	LockRetryInterval:     10 * time.Millisecond,
	LockMaxRetries:        300,
	LockMaxInterval:       500 * time.Millisecond,
	LockBackoffMultiplier: 2.0,
}

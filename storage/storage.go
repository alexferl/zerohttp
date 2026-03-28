package storage

import (
	"context"
	"errors"
	"time"
)

// ErrLockNotSupported is returned by NewStorageAdapter when the provided
// storage backend does not implement storage.Locker.
var ErrLockNotSupported = errors.New("storage: locking not supported by backend")

// Storage is the low-level key-value interface for storage backends.
// Users implement this interface to provide their own storage (Redis, PostgreSQL, etc.)
type Storage interface {
	// Get retrieves a value by key.
	// Returns the value, true if found, and any error.
	// If not found, returns nil, false, nil error.
	Get(ctx context.Context, key string) ([]byte, bool, error)

	// Set stores a value with the given TTL.
	// Returns an error if the operation fails.
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error

	// Delete removes a key from storage.
	// Returns an error if the operation fails.
	Delete(ctx context.Context, key string) error

	// Close releases resources associated with the storage.
	// Returns an error if the close operation fails.
	Close() error
}

// Locker is an optional interface for Storage implementations that support
// distributed locking (required by idempotency middleware).
type Locker interface {
	// Lock acquires an exclusive lock for the given key with the specified TTL.
	// The TTL ensures the lock auto-expires if the caller crashes between Lock
	// and Unlock, preventing permanent deadlocks.
	// Returns true if the lock was acquired, false if already locked.
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Unlock releases the lock for the given key.
	// Implementations SHOULD verify ownership before releasing (e.g., via a
	// token stored at Lock time) to prevent accidental release of another
	// holder's lock if the TTL expired. A plain DELETE is not safe.
	Unlock(ctx context.Context, key string) error
}

// Inspector is an optional interface for Storage implementations that support
// TTL introspection. Not used by current middleware adapters, but available
// for custom Store implementations that need stale-while-revalidate behavior.
type Inspector interface {
	// TTL returns the remaining time-to-live for a key.
	// Returns 0 if the key does not exist or has no TTL.
	TTL(ctx context.Context, key string) (time.Duration, error)
}

package idempotency

import (
	"context"
	"encoding/json"
	"time"

	"github.com/alexferl/zerohttp/storage"
)

// StorageAdapterConfig configures the StorageAdapter.
type StorageAdapterConfig struct {
	// LockTTL is the TTL for distributed locks.
	// Default: 30s
	LockTTL time.Duration
}

// DefaultStorageAdapterConfig is the default configuration for StorageAdapter.
var DefaultStorageAdapterConfig = StorageAdapterConfig{
	LockTTL: 30 * time.Second,
}

// StorageAdapter wraps a storage.Storage to implement the idempotency.Store interface.
type StorageAdapter struct {
	store   storage.Storage
	locker  storage.Locker
	lockTTL time.Duration
}

// NewStorageAdapter creates an idempotency.Store from a storage.Storage.
// Returns an error if the storage does not implement storage.Locker.
func NewStorageAdapter(s storage.Storage, cfg ...StorageAdapterConfig) (Store, error) {
	locker, ok := s.(storage.Locker)
	if !ok {
		return nil, storage.ErrLockNotSupported
	}

	c := DefaultStorageAdapterConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	return &StorageAdapter{
		store:   s,
		locker:  locker,
		lockTTL: c.LockTTL,
	}, nil
}

// Get retrieves a cached response by key.
func (a *StorageAdapter) Get(ctx context.Context, key string) (Record, bool, error) {
	data, found, err := a.store.Get(ctx, key)
	if !found || err != nil {
		return Record{}, false, err
	}

	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, false, err
	}

	return rec, true, nil
}

// Set stores a response in the cache with the given TTL.
func (a *StorageAdapter) Set(ctx context.Context, key string, rec Record, ttl time.Duration) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	return a.store.Set(ctx, key, data, ttl)
}

// Close releases resources associated with the underlying storage.
func (a *StorageAdapter) Close() error {
	return a.store.Close()
}

// Lock acquires an exclusive lock for the given key.
func (a *StorageAdapter) Lock(ctx context.Context, key string) (bool, error) {
	return a.locker.Lock(ctx, key, a.lockTTL)
}

// Unlock releases the lock for the given key.
func (a *StorageAdapter) Unlock(ctx context.Context, key string) error {
	return a.locker.Unlock(ctx, key)
}

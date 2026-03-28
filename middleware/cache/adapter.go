package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/alexferl/zerohttp/storage"
)

// StorageAdapter wraps a storage.Storage to implement the cache.Store interface.
type StorageAdapter struct {
	store storage.Storage
}

// NewStorageAdapter creates a cache.Store from a storage.Storage.
func NewStorageAdapter(s storage.Storage) Store {
	return &StorageAdapter{store: s}
}

// Get retrieves a cached record by key.
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

// Set stores a record in the cache with the given TTL.
func (a *StorageAdapter) Set(ctx context.Context, key string, rec Record, ttl time.Duration) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}

	return a.store.Set(ctx, key, data, ttl)
}

// Delete removes a cached record by key.
func (a *StorageAdapter) Delete(ctx context.Context, key string) error {
	return a.store.Delete(ctx, key)
}

// Close releases resources associated with the underlying storage.
func (a *StorageAdapter) Close() error {
	return a.store.Close()
}

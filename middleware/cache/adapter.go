package cache

import (
	"context"
	"encoding/json"
	"time"

	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/storage"
)

// Codec handles serialization and deserialization of cache records.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// JSONCodec is the default codec using encoding/json.
type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (JSONCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// StorageAdapterConfig configures the StorageAdapter.
type StorageAdapterConfig struct {
	// KeyPrefix is the prefix for cache keys.
	// Default: "cache:"
	KeyPrefix string

	// Codec is the serialization codec for cache records.
	// Default: JSONCodec
	Codec Codec
}

// DefaultStorageAdapterConfig is the default configuration for StorageAdapter.
var DefaultStorageAdapterConfig = StorageAdapterConfig{
	KeyPrefix: "cache:",
	Codec:     JSONCodec{},
}

// StorageAdapter wraps a storage.Storage to implement the cache.Store interface.
type StorageAdapter struct {
	store     storage.Storage
	keyPrefix string
	codec     Codec
}

// NewStorageAdapter creates a cache.Store from a storage.Storage.
func NewStorageAdapter(s storage.Storage, cfg ...StorageAdapterConfig) Store {
	c := DefaultStorageAdapterConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}
	return &StorageAdapter{
		store:     s,
		keyPrefix: c.KeyPrefix,
		codec:     c.Codec,
	}
}

func (a *StorageAdapter) prefixKey(key string) string {
	return a.keyPrefix + key
}

// Get retrieves a cached record by key.
func (a *StorageAdapter) Get(ctx context.Context, key string) (Record, bool, error) {
	data, found, err := a.store.Get(ctx, a.prefixKey(key))
	if !found || err != nil {
		return Record{}, false, err
	}

	var rec Record
	if err := a.codec.Unmarshal(data, &rec); err != nil {
		return Record{}, false, err
	}

	return rec, true, nil
}

// Set stores a record in the cache with the given TTL.
func (a *StorageAdapter) Set(ctx context.Context, key string, rec Record, ttl time.Duration) error {
	data, err := a.codec.Marshal(rec)
	if err != nil {
		return err
	}

	return a.store.Set(ctx, a.prefixKey(key), data, ttl)
}

// Delete removes a cached record by key.
func (a *StorageAdapter) Delete(ctx context.Context, key string) error {
	return a.store.Delete(ctx, a.prefixKey(key))
}

// Close releases resources associated with the underlying storage.
func (a *StorageAdapter) Close() error {
	return a.store.Close()
}

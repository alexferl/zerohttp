package jwtauth

import (
	"context"
	"fmt"
	"time"

	"github.com/alexferl/zerohttp/storage"
)

// StorageAdapter wraps a storage.Storage to implement RevocationStore.
// This allows jwtauth to use any storage.Storage backend for token revocation.
type StorageAdapter struct {
	storage   storage.Storage
	keyPrefix string
}

// StorageAdapterConfig configures the StorageAdapter.
type StorageAdapterConfig struct {
	// KeyPrefix is the prefix for revocation keys.
	// Default: "jwt:revoke:"
	KeyPrefix string
}

// DefaultStorageAdapterConfig is the default configuration for StorageAdapter.
var DefaultStorageAdapterConfig = StorageAdapterConfig{
	KeyPrefix: "jwt:revoke:",
}

// NewStorageAdapter creates a RevocationStore from a storage.Storage.
// This allows sharing the same storage backend between idempotency, cache, and jwtauth.
//
// Example:
//
//	redisStorage := storage.NewRedisStorage(redisClient, storage.RedisStorageConfig{
//	    KeyPrefix: "app:",
//	})
//	revocationStore := jwtauth.NewStorageAdapter(redisStorage)
//
//	// Use with HS256Store for revocation support
//	store := jwtauth.NewHS256StoreWithRevocation(secret, jwtauth.HS256Config{}, revocationStore)
//
//	// Or use with a custom Store
//	type myStore struct {
//	    jwtauth.RevocationStore
//	    // ... other fields
//	}
func NewStorageAdapter(s storage.Storage, cfg ...StorageAdapterConfig) RevocationStore {
	c := DefaultStorageAdapterConfig
	if len(cfg) > 0 {
		if cfg[0].KeyPrefix != "" {
			c.KeyPrefix = cfg[0].KeyPrefix
		}
	}

	return &StorageAdapter{
		storage:   s,
		keyPrefix: c.KeyPrefix,
	}
}

func (a *StorageAdapter) revokeKey(jti string) string {
	return a.keyPrefix + jti
}

// Revoke stores a revocation marker for the given JTI with the specified TTL.
// The marker value is a simple "1" to minimize storage.
func (a *StorageAdapter) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	return a.storage.Set(ctx, a.revokeKey(jti), []byte("1"), ttl)
}

// IsRevoked checks if a revocation marker exists for the given JTI.
func (a *StorageAdapter) IsRevoked(ctx context.Context, jti string) (bool, error) {
	_, found, err := a.storage.Get(ctx, a.revokeKey(jti))
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return found, nil
}

// Close releases resources associated with the underlying storage.
func (a *StorageAdapter) Close() error {
	return a.storage.Close()
}

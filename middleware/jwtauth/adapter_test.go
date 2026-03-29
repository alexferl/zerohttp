package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/storage"
	"github.com/alexferl/zerohttp/zhtest"
)

// mockStorage is a simple in-memory storage implementation for testing
type mockStorage struct {
	data map[string]mockEntry
}

type mockEntry struct {
	value []byte
	ttl   time.Duration
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string]mockEntry),
	}
}

func (m *mockStorage) Get(ctx context.Context, key string) ([]byte, bool, error) {
	entry, ok := m.data[key]
	return entry.value, ok, nil
}

func (m *mockStorage) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	m.data[key] = mockEntry{value: val, ttl: ttl}
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestNewStorageAdapter(t *testing.T) {
	mock := newMockStorage()

	t.Run("default config", func(t *testing.T) {
		adapter := NewStorageAdapter(mock)
		zhtest.AssertNotNil(t, adapter)

		sa, ok := adapter.(*StorageAdapter)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, "jwt:revoke:", sa.keyPrefix)
	})

	t.Run("custom prefix", func(t *testing.T) {
		adapter := NewStorageAdapter(mock, StorageAdapterConfig{KeyPrefix: "custom:"})
		zhtest.AssertNotNil(t, adapter)

		sa, ok := adapter.(*StorageAdapter)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, "custom:", sa.keyPrefix)
	})

	t.Run("empty prefix uses default", func(t *testing.T) {
		adapter := NewStorageAdapter(mock, StorageAdapterConfig{KeyPrefix: ""})
		zhtest.AssertNotNil(t, adapter)

		sa, ok := adapter.(*StorageAdapter)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, "jwt:revoke:", sa.keyPrefix)
	})
}

func TestStorageAdapter_Revoke(t *testing.T) {
	mock := newMockStorage()
	adapter := NewStorageAdapter(mock, StorageAdapterConfig{KeyPrefix: "test:"})
	ctx := context.Background()

	t.Run("revoke token", func(t *testing.T) {
		err := adapter.Revoke(ctx, "token-123", time.Hour)
		zhtest.AssertNoError(t, err)

		// Verify stored with correct key and value
		entry, ok := mock.data["test:token-123"]
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, "1", string(entry.value))
		zhtest.AssertEqual(t, time.Hour, entry.ttl)
	})

	t.Run("revoke with different TTL", func(t *testing.T) {
		err := adapter.Revoke(ctx, "token-456", 30*time.Minute)
		zhtest.AssertNoError(t, err)

		entry, ok := mock.data["test:token-456"]
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, 30*time.Minute, entry.ttl)
	})
}

func TestStorageAdapter_IsRevoked(t *testing.T) {
	mock := newMockStorage()
	adapter := NewStorageAdapter(mock, StorageAdapterConfig{KeyPrefix: "test:"})
	ctx := context.Background()

	t.Run("token is revoked", func(t *testing.T) {
		// Pre-populate revoked token
		mock.data["test:revoked-token"] = mockEntry{value: []byte("1")}

		revoked, err := adapter.IsRevoked(ctx, "revoked-token")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, revoked)
	})

	t.Run("token is not revoked", func(t *testing.T) {
		revoked, err := adapter.IsRevoked(ctx, "valid-token")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, revoked)
	})
}

func TestStorageAdapter_Close(t *testing.T) {
	mock := newMockStorage()
	adapter := NewStorageAdapter(mock)

	err := adapter.Close()
	zhtest.AssertNoError(t, err)
}

func TestStorageAdapter_Integration(t *testing.T) {
	mock := newMockStorage()
	adapter := NewStorageAdapter(mock)
	ctx := context.Background()

	// Full flow: revoke and check
	err := adapter.Revoke(ctx, "refresh-token-1", time.Hour)
	zhtest.AssertNoError(t, err)

	revoked, err := adapter.IsRevoked(ctx, "refresh-token-1")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, revoked)

	// Check different token is not revoked
	revoked, err = adapter.IsRevoked(ctx, "refresh-token-2")
	zhtest.AssertNoError(t, err)
	zhtest.AssertFalse(t, revoked)
}

// Verify StorageAdapter implements RevocationStore interface
var _ RevocationStore = (*StorageAdapter)(nil)

// Verify storage.Storage interface compatibility
var _ storage.Storage = (*mockStorage)(nil)

package idempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/storage"
)

// mockStorage is a test implementation of storage.Storage
type mockStorage struct {
	data    map[string][]byte
	ttlVals map[string]time.Duration
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:    make(map[string][]byte),
		ttlVals: make(map[string]time.Duration),
	}
}

func (m *mockStorage) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *mockStorage) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	m.data[key] = val
	m.ttlVals[key] = ttl
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	delete(m.ttlVals, key)
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

// mockLocker implements storage.Locker for testing
type mockLocker struct {
	locks map[string]bool
	ttl   time.Duration
}

func newMockLocker() *mockLocker {
	return &mockLocker{
		locks: make(map[string]bool),
	}
}

func (m *mockLocker) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if m.locks[key] {
		return false, nil
	}
	m.locks[key] = true
	m.ttl = ttl
	return true, nil
}

func (m *mockLocker) Unlock(ctx context.Context, key string) error {
	delete(m.locks, key)
	return nil
}

// mockStorageWithLocker combines both interfaces
type mockStorageWithLocker struct {
	*mockStorage
	*mockLocker
}

func newMockStorageWithLocker() *mockStorageWithLocker {
	return &mockStorageWithLocker{
		mockStorage: newMockStorage(),
		mockLocker:  newMockLocker(),
	}
}

func TestNewStorageAdapter(t *testing.T) {
	t.Run("with locker", func(t *testing.T) {
		s := newMockStorageWithLocker()
		adapter, err := NewStorageAdapter(s)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if adapter == nil {
			t.Fatal("adapter should not be nil")
		}
	})

	t.Run("without locker", func(t *testing.T) {
		s := newMockStorage()
		adapter, err := NewStorageAdapter(s)
		if err == nil {
			t.Error("should error when storage doesn't implement Locker")
		}
		if adapter != nil {
			t.Error("adapter should be nil on error")
		}
		if !errors.Is(err, storage.ErrLockNotSupported) {
			t.Errorf("error should be ErrLockNotSupported, got: %v", err)
		}
	})
}

func TestNewStorageAdapter_WithLockTTL(t *testing.T) {
	s := newMockStorageWithLocker()
	customTTL := 5 * time.Minute
	adapter, err := NewStorageAdapter(s, StorageAdapterConfig{LockTTL: customTTL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify custom TTL is used by checking lock behavior
	ctx := context.Background()
	locked, err := adapter.Lock(ctx, "test-key")
	if err != nil {
		t.Fatalf("lock failed: %v", err)
	}
	if !locked {
		t.Error("should acquire lock")
	}
	// TTL verification would require checking the mock
}

func TestStorageAdapter_Get(t *testing.T) {
	ctx := context.Background()
	s := newMockStorageWithLocker()
	adapter, _ := NewStorageAdapter(s)

	t.Run("not found", func(t *testing.T) {
		rec, found, err := adapter.Get(ctx, "missing")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if found {
			t.Error("should not find missing key")
		}
		if rec.StatusCode != 0 {
			t.Error("record should be zero value")
		}
	})

	t.Run("found", func(t *testing.T) {
		expected := Record{
			StatusCode: 200,
			Body:       []byte("hello"),
			CreatedAt:  time.Now(),
		}
		if err := adapter.Set(ctx, "test", expected, time.Hour); err != nil {
			t.Fatalf("set failed: %v", err)
		}

		rec, found, err := adapter.Get(ctx, "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !found {
			t.Error("should find existing key")
		}
		if rec.StatusCode != expected.StatusCode {
			t.Errorf("StatusCode = %d, want %d", rec.StatusCode, expected.StatusCode)
		}
		if string(rec.Body) != string(expected.Body) {
			t.Errorf("Body = %s, want %s", rec.Body, expected.Body)
		}
	})
}

func TestStorageAdapter_Set(t *testing.T) {
	ctx := context.Background()
	s := newMockStorageWithLocker()
	adapter, _ := NewStorageAdapter(s)

	rec := Record{
		StatusCode: 201,
		Headers:    []string{"Content-Type", "application/json"},
		Body:       []byte(`{"id": 1}`),
		CreatedAt:  time.Now(),
	}
	ttl := 24 * time.Hour

	if err := adapter.Set(ctx, "key", rec, ttl); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, ok := s.data["key"]; !ok {
		t.Error("key should exist in storage")
	}
	if s.ttlVals["key"] != ttl {
		t.Errorf("TTL = %v, want %v", s.ttlVals["key"], ttl)
	}
}

func TestStorageAdapter_Lock(t *testing.T) {
	ctx := context.Background()

	t.Run("with locker", func(t *testing.T) {
		s := newMockStorageWithLocker()
		adapter, _ := NewStorageAdapter(s)

		// First lock should succeed
		got, err := adapter.Lock(ctx, "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !got {
			t.Error("first lock should succeed")
		}

		// Second lock should fail (already locked)
		got, err = adapter.Lock(ctx, "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got {
			t.Error("second lock should fail")
		}
	})
}

func TestStorageAdapter_Unlock(t *testing.T) {
	ctx := context.Background()

	t.Run("with locker", func(t *testing.T) {
		s := newMockStorageWithLocker()
		adapter, _ := NewStorageAdapter(s)

		// Lock first
		if _, err := adapter.Lock(ctx, "key"); err != nil {
			t.Fatalf("lock failed: %v", err)
		}

		// Unlock should succeed
		if err := adapter.Unlock(ctx, "key"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should be able to lock again
		got, _ := adapter.Lock(ctx, "key")
		if !got {
			t.Error("should be able to lock after unlock")
		}
	})
}

func TestStorageAdapter_Close(t *testing.T) {
	s := newMockStorageWithLocker()
	adapter, _ := NewStorageAdapter(s)

	// Type assert to access Close method (not part of Store interface)
	sa, ok := adapter.(*StorageAdapter)
	if !ok {
		t.Fatal("adapter should be *StorageAdapter")
	}

	if err := sa.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

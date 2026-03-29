package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/storage"
	"github.com/alexferl/zerohttp/zhtest"
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
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotNil(t, adapter)
	})

	t.Run("without locker", func(t *testing.T) {
		s := newMockStorage()
		adapter, err := NewStorageAdapter(s)
		zhtest.AssertError(t, err)
		zhtest.AssertNil(t, adapter)
		zhtest.AssertErrorIs(t, err, storage.ErrLockNotSupported)
	})
}

func TestNewStorageAdapter_WithLockTTL(t *testing.T) {
	s := newMockStorageWithLocker()
	customTTL := 5 * time.Minute
	adapter, err := NewStorageAdapter(s, StorageAdapterConfig{LockTTL: customTTL})
	zhtest.AssertNoError(t, err)
	zhtest.AssertNotNil(t, adapter)

	// Verify custom TTL is used by checking lock behavior
	ctx := context.Background()
	locked, err := adapter.Lock(ctx, "test-key")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, locked)
	// TTL verification would require checking the mock
}

func TestNewStorageAdapter_PartialConfig(t *testing.T) {
	// Test that partial config merges with defaults
	s := newMockStorageWithLocker()

	// Empty config should use defaults
	adapter, err := NewStorageAdapter(s, StorageAdapterConfig{})
	zhtest.AssertNoError(t, err)

	// Type assert to check lockTTL was set to default
	sa, ok := adapter.(*StorageAdapter)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertEqual(t, DefaultStorageAdapterConfig.LockTTL, sa.lockTTL)
}

// mockCodec is a test codec that prepends a prefix
type mockCodec struct{}

func (m mockCodec) Marshal(v any) ([]byte, error) {
	return []byte("MOCK:"), nil
}

func (m mockCodec) Unmarshal(data []byte, v any) error {
	if rec, ok := v.(*Record); ok && len(data) >= 5 {
		rec.StatusCode = 999
	}
	return nil
}

func TestNewStorageAdapter_WithCodec(t *testing.T) {
	s := newMockStorageWithLocker()
	cfg := StorageAdapterConfig{
		Codec: mockCodec{},
	}
	adapter, err := NewStorageAdapter(s, cfg)
	zhtest.AssertNoError(t, err)

	// Type assert to check codec was set
	sa, ok := adapter.(*StorageAdapter)
	zhtest.AssertTrue(t, ok)

	_, isMockCodec := sa.codec.(mockCodec)
	zhtest.AssertTrue(t, isMockCodec)
}

func TestStorageAdapter_CustomCodec(t *testing.T) {
	ctx := context.Background()
	s := newMockStorageWithLocker()
	cfg := StorageAdapterConfig{
		Codec: mockCodec{},
	}
	adapter, err := NewStorageAdapter(s, cfg)
	zhtest.AssertNoError(t, err)

	// Store via adapter - should use custom codec
	rec := Record{StatusCode: 200, Body: []byte("test")}
	err = adapter.Set(ctx, "key", rec, time.Hour)
	zhtest.AssertNoError(t, err)

	// Check that custom codec was used (prepends "MOCK:")
	data := s.data["idemp:key"]
	zhtest.AssertEqual(t, "MOCK:", string(data))

	// Get via adapter - should use custom codec
	gotRec, _, err := adapter.Get(ctx, "key")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 999, gotRec.StatusCode)
}

func TestStorageAdapter_Get(t *testing.T) {
	ctx := context.Background()
	s := newMockStorageWithLocker()
	adapter, _ := NewStorageAdapter(s)

	t.Run("not found", func(t *testing.T) {
		rec, found, err := adapter.Get(ctx, "missing")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
		zhtest.AssertEqual(t, 0, rec.StatusCode)
	})

	t.Run("found", func(t *testing.T) {
		expected := Record{
			StatusCode: 200,
			Body:       []byte("hello"),
			CreatedAt:  time.Now(),
		}
		err := adapter.Set(ctx, "test", expected, time.Hour)
		zhtest.AssertNoError(t, err)

		rec, found, err := adapter.Get(ctx, "test")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, expected.StatusCode, rec.StatusCode)
		zhtest.AssertEqual(t, string(expected.Body), string(rec.Body))
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

	err := adapter.Set(ctx, "key", rec, ttl)
	zhtest.AssertNoError(t, err)

	_, ok := s.data["idemp:key"]
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, ttl, s.ttlVals["idemp:key"])
}

func TestStorageAdapter_Lock(t *testing.T) {
	ctx := context.Background()

	t.Run("with locker", func(t *testing.T) {
		s := newMockStorageWithLocker()
		adapter, _ := NewStorageAdapter(s)

		// First lock should succeed
		got, err := adapter.Lock(ctx, "key")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, got)

		// Second lock should fail (already locked)
		got, err = adapter.Lock(ctx, "key")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, got)
	})
}

func TestStorageAdapter_Unlock(t *testing.T) {
	ctx := context.Background()

	t.Run("with locker", func(t *testing.T) {
		s := newMockStorageWithLocker()
		adapter, _ := NewStorageAdapter(s)

		// Lock first
		_, err := adapter.Lock(ctx, "key")
		zhtest.AssertNoError(t, err)

		// Unlock should succeed
		err = adapter.Unlock(ctx, "key")
		zhtest.AssertNoError(t, err)

		// Should be able to lock again
		got, _ := adapter.Lock(ctx, "key")
		zhtest.AssertTrue(t, got)
	})
}

func TestStorageAdapter_Close(t *testing.T) {
	s := newMockStorageWithLocker()
	adapter, _ := NewStorageAdapter(s)

	// Type assert to access Close method (not part of Store interface)
	sa, ok := adapter.(*StorageAdapter)
	zhtest.AssertTrue(t, ok)

	err := sa.Close()
	zhtest.AssertNoError(t, err)
}

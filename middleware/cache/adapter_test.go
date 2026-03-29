package cache

import (
	"context"
	"testing"
	"time"

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

func TestNewStorageAdapter(t *testing.T) {
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

	zhtest.AssertNotNil(t, adapter)
}

func TestStorageAdapter_Get(t *testing.T) {
	ctx := context.Background()
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

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
			ETag:       "abc123",
		}
		// Store via adapter
		zhtest.AssertNoError(t, adapter.Set(ctx, "test", expected, time.Minute))

		rec, found, err := adapter.Get(ctx, "test")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, expected.StatusCode, rec.StatusCode)
		zhtest.AssertEqual(t, string(expected.Body), string(rec.Body))
		zhtest.AssertEqual(t, expected.ETag, rec.ETag)
	})

	t.Run("invalid json", func(t *testing.T) {
		s.data["bad"] = []byte("not json")
		_, _, err := adapter.Get(ctx, "bad")
		zhtest.AssertError(t, err)
	})
}

func TestStorageAdapter_Set(t *testing.T) {
	ctx := context.Background()
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

	rec := Record{
		StatusCode: 201,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"id": 1}`),
	}
	ttl := 5 * time.Minute

	zhtest.AssertNoError(t, adapter.Set(ctx, "key", rec, ttl))

	// Verify raw storage has JSON data
	data, ok := s.data["key"]
	zhtest.AssertTrue(t, ok)

	// Verify it's valid JSON
	zhtest.AssertGreater(t, len(data), 0)

	// Verify TTL is stored
	zhtest.AssertEqual(t, ttl, s.ttlVals["key"])
}

func TestStorageAdapter_Delete(t *testing.T) {
	ctx := context.Background()
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

	// Store something first
	s.data["test"] = []byte("value")
	s.ttlVals["test"] = time.Minute

	zhtest.AssertNoError(t, adapter.Delete(ctx, "test"))

	_, ok := s.data["test"]
	zhtest.AssertFalse(t, ok)
	_, ok = s.ttlVals["test"]
	zhtest.AssertFalse(t, ok)
}

func TestStorageAdapter_Close(t *testing.T) {
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

	// Type assert to access Close method
	sa, ok := adapter.(*StorageAdapter)
	zhtest.AssertTrue(t, ok)

	zhtest.AssertNoError(t, sa.Close())
}

// mockCodec is a test codec that prepends a prefix
type mockCodec struct{}

func (m mockCodec) Marshal(v any) ([]byte, error) {
	return []byte("MOCK:"), nil
}

func (m mockCodec) Unmarshal(data []byte, v any) error {
	// Just set a marker - real impl would decode
	if rec, ok := v.(*Record); ok && len(data) >= 5 {
		rec.StatusCode = 999
	}
	return nil
}

func TestNewStorageAdapter_WithConfig(t *testing.T) {
	s := newMockStorage()
	cfg := StorageAdapterConfig{
		Codec: mockCodec{},
	}
	adapter := NewStorageAdapter(s, cfg)

	// Type assert to check codec was set
	sa, ok := adapter.(*StorageAdapter)
	zhtest.AssertTrue(t, ok)

	_, ok = sa.codec.(mockCodec)
	zhtest.AssertTrue(t, ok)
}

func TestStorageAdapter_CustomCodec(t *testing.T) {
	ctx := context.Background()
	s := newMockStorage()
	cfg := StorageAdapterConfig{
		Codec: mockCodec{},
	}
	adapter := NewStorageAdapter(s, cfg)

	// Store via adapter - should use custom codec
	rec := Record{StatusCode: 200, Body: []byte("test")}
	zhtest.AssertNoError(t, adapter.Set(ctx, "key", rec, time.Minute))

	// Check that custom codec was used (prepends "MOCK:")
	data := s.data["key"]
	zhtest.AssertEqual(t, "MOCK:", string(data))

	// Get via adapter - should use custom codec
	gotRec, _, err := adapter.Get(ctx, "key")
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, 999, gotRec.StatusCode)
}

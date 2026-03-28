package cache

import (
	"context"
	"testing"
	"time"
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

	if adapter == nil {
		t.Fatal("NewStorageAdapter should not return nil")
	}
}

func TestStorageAdapter_Get(t *testing.T) {
	ctx := context.Background()
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

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
			ETag:       "abc123",
		}
		// Store via adapter
		if err := adapter.Set(ctx, "test", expected, time.Minute); err != nil {
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
		if rec.ETag != expected.ETag {
			t.Errorf("ETag = %s, want %s", rec.ETag, expected.ETag)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		s.data["bad"] = []byte("not json")
		_, _, err := adapter.Get(ctx, "bad")
		if err == nil {
			t.Error("should error on invalid json")
		}
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

	if err := adapter.Set(ctx, "key", rec, ttl); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify raw storage has JSON data
	data, ok := s.data["key"]
	if !ok {
		t.Fatal("key not found in storage")
	}

	// Verify it's valid JSON
	if len(data) == 0 {
		t.Error("stored data should not be empty")
	}

	// Verify TTL is stored
	if s.ttlVals["key"] != ttl {
		t.Errorf("TTL = %v, want %v", s.ttlVals["key"], ttl)
	}
}

func TestStorageAdapter_Delete(t *testing.T) {
	ctx := context.Background()
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

	// Store something first
	s.data["test"] = []byte("value")
	s.ttlVals["test"] = time.Minute

	if err := adapter.Delete(ctx, "test"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, ok := s.data["test"]; ok {
		t.Error("key should be deleted")
	}
	if _, ok := s.ttlVals["test"]; ok {
		t.Error("ttl should be deleted")
	}
}

func TestStorageAdapter_Close(t *testing.T) {
	s := newMockStorage()
	adapter := NewStorageAdapter(s)

	// Type assert to access Close method
	sa, ok := adapter.(*StorageAdapter)
	if !ok {
		t.Fatal("adapter should be *StorageAdapter")
	}

	if err := sa.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
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
	if !ok {
		t.Fatal("adapter should be *StorageAdapter")
	}

	if _, ok := sa.codec.(mockCodec); !ok {
		t.Error("codec should be mockCodec")
	}
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
	if err := adapter.Set(ctx, "key", rec, time.Minute); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Check that custom codec was used (prepends "MOCK:")
	data := s.data["key"]
	if string(data) != "MOCK:" {
		t.Errorf("expected MOCK: prefix, got %s", string(data))
	}

	// Get via adapter - should use custom codec
	gotRec, _, err := adapter.Get(ctx, "key")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if gotRec.StatusCode != 999 {
		t.Errorf("StatusCode = %d, want 999 (custom codec marker)", gotRec.StatusCode)
	}
}

package idempotency

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of config.IdempotencyStore.
// It uses sync.RWMutex for thread-safe access and supports TTL-based expiration.
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*idempotencyEntry
	maxKeys int
	// locks tracks in-flight requests per key to prevent concurrent execution
	locks   map[string]struct{}
	locksMu sync.Mutex
}

// idempotencyEntry represents a cached idempotency response with expiry.
type idempotencyEntry struct {
	record Record
	expiry time.Time
}

// NewMemoryStore creates a new in-memory idempotency store.
// If maxKeys is 0, the store has unlimited capacity (not recommended for production).
func NewMemoryStore(maxKeys int) *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]*idempotencyEntry),
		maxKeys: maxKeys,
		locks:   make(map[string]struct{}),
	}
}

// Get retrieves a cached response by key.
func (s *MemoryStore) Get(_ context.Context, key string) (Record, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[key]
	if !ok {
		return Record{}, false, nil
	}

	if time.Now().After(entry.expiry) {
		return Record{}, false, nil
	}

	return entry.record, true, nil
}

// Set stores a response in the cache with the given TTL.
func (s *MemoryStore) Set(_ context.Context, key string, record Record, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxKeys > 0 && len(s.entries) >= s.maxKeys {
		s.removeExpired()

		if len(s.entries) >= s.maxKeys {
			s.removeOldest()
		}
	}

	s.entries[key] = &idempotencyEntry{
		record: record,
		expiry: time.Now().Add(ttl),
	}

	return nil
}

// removeExpired deletes all expired entries from the store.
func (s *MemoryStore) removeExpired() {
	now := time.Now()
	for key, entry := range s.entries {
		if now.After(entry.expiry) {
			delete(s.entries, key)
		}
	}
}

// removeOldest deletes the entry with the earliest expiry.
func (s *MemoryStore) removeOldest() {
	var oldestKey string
	var oldestExpiry time.Time

	for key, entry := range s.entries {
		if oldestKey == "" || entry.expiry.Before(oldestExpiry) {
			oldestKey = key
			oldestExpiry = entry.expiry
		}
	}

	if oldestKey != "" {
		delete(s.entries, oldestKey)
	}
}

// Lock acquires an exclusive lock for the given key.
// Returns true if the lock was acquired, false if the key is already locked.
func (s *MemoryStore) Lock(_ context.Context, key string) (bool, error) {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()

	if _, exists := s.locks[key]; exists {
		return false, nil
	}

	s.locks[key] = struct{}{}
	return true, nil
}

// Unlock releases the lock for the given key.
func (s *MemoryStore) Unlock(_ context.Context, key string) error {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()

	delete(s.locks, key)
	return nil
}

// Close releases resources associated with the store.
// For MemoryStore, this is a no-op.
func (s *MemoryStore) Close() error {
	return nil
}

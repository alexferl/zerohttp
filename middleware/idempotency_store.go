package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// IdempotencyMemoryStore is an in-memory implementation of config.IdempotencyStore.
// It uses sync.RWMutex for thread-safe access and supports TTL-based expiration.
type IdempotencyMemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*idempotencyEntry
	maxKeys int
	// locks tracks in-flight requests per key to prevent concurrent execution
	locks   map[string]struct{}
	locksMu sync.Mutex
}

// idempotencyEntry represents a cached idempotency response with expiry.
type idempotencyEntry struct {
	record config.IdempotencyRecord
	expiry time.Time
}

// NewIdempotencyMemoryStore creates a new in-memory idempotency store.
// If maxKeys is 0, the store has unlimited capacity (not recommended for production).
func NewIdempotencyMemoryStore(maxKeys int) *IdempotencyMemoryStore {
	return &IdempotencyMemoryStore{
		entries: make(map[string]*idempotencyEntry),
		maxKeys: maxKeys,
		locks:   make(map[string]struct{}),
	}
}

// Get retrieves a cached response by key.
func (s *IdempotencyMemoryStore) Get(_ context.Context, key string) (config.IdempotencyRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[key]
	if !ok {
		return config.IdempotencyRecord{}, false, nil
	}

	if time.Now().After(entry.expiry) {
		return config.IdempotencyRecord{}, false, nil
	}

	return entry.record, true, nil
}

// Set stores a response in the cache with the given TTL.
func (s *IdempotencyMemoryStore) Set(_ context.Context, key string, record config.IdempotencyRecord, ttl time.Duration) error {
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
func (s *IdempotencyMemoryStore) removeExpired() {
	now := time.Now()
	for key, entry := range s.entries {
		if now.After(entry.expiry) {
			delete(s.entries, key)
		}
	}
}

// removeOldest deletes the entry with the earliest expiry.
func (s *IdempotencyMemoryStore) removeOldest() {
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
func (s *IdempotencyMemoryStore) Lock(_ context.Context, key string) (bool, error) {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()

	if _, exists := s.locks[key]; exists {
		return false, nil
	}

	s.locks[key] = struct{}{}
	return true, nil
}

// Unlock releases the lock for the given key.
func (s *IdempotencyMemoryStore) Unlock(_ context.Context, key string) error {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()

	delete(s.locks, key)
	return nil
}

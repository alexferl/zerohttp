package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// cacheEntry represents a single cached entry with expiry and LRU tracking.
type cacheEntry struct {
	key        string
	record     Record
	expiry     time.Time
	lruElement *list.Element
}

// MemoryStore is a thread-safe in-memory cache with LRU eviction.
type MemoryStore struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	lruList    *list.List
	lruIndex   map[string]*list.Element
	maxEntries int
}

// NewMemoryStore creates a new in-memory cache store.
// If maxEntries is 0, the cache has unlimited capacity (not recommended for production).
func NewMemoryStore(maxEntries int) *MemoryStore {
	return &MemoryStore{
		entries:    make(map[string]*cacheEntry),
		lruList:    list.New(),
		lruIndex:   make(map[string]*list.Element),
		maxEntries: maxEntries,
	}
}

// Get retrieves a cached entry by key.
// Returns the record, true if found and not expired, and nil error.
// Returns false and nil error if not found or expired.
// The context is accepted for interface compatibility but not used by the in-memory store.
func (c *MemoryStore) Get(_ context.Context, key string) (Record, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return Record{}, false, nil
	}

	if time.Now().After(entry.expiry) {
		c.removeEntry(entry)
		return Record{}, false, nil
	}

	c.lruList.MoveToFront(entry.lruElement)

	return entry.record, true, nil
}

// Set stores a record in the cache with the given TTL.
// Returns nil error on success.
// The context is accepted for interface compatibility but not used by the in-memory store.
func (c *MemoryStore) Set(_ context.Context, key string, record Record, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.lruIndex[key]; exists {
		c.lruList.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.record = record
		entry.expiry = time.Now().Add(ttl)
		return nil
	}

	if c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		c.evictOldest()
	}

	entry := &cacheEntry{
		key:    key,
		record: record,
		expiry: time.Now().Add(ttl),
	}
	elem := c.lruList.PushFront(entry)
	entry.lruElement = elem
	c.lruIndex[key] = elem
	c.entries[key] = entry
	return nil
}

// removeEntry removes an entry from all internal data structures.
func (c *MemoryStore) removeEntry(entry *cacheEntry) {
	delete(c.entries, entry.key)
	delete(c.lruIndex, entry.key)
	c.lruList.Remove(entry.lruElement)
}

// evictOldest removes the least recently used entry.
func (c *MemoryStore) evictOldest() {
	if elem := c.lruList.Back(); elem != nil {
		entry := elem.Value.(*cacheEntry)
		c.removeEntry(entry)
	}
}

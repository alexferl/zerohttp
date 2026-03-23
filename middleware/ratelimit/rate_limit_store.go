package ratelimit

import (
	"context"
	"sync"
	"time"
)

// bucketEntry represents a token bucket for rate limiting.
type bucketEntry struct {
	tokens     float64
	capacity   float64
	rate       float64
	lastRefill time.Time
	lastAccess time.Time
	mutex      sync.Mutex
}

// counterEntry represents a fixed window counter.
type counterEntry struct {
	count       int
	windowStart time.Time
	lastAccess  time.Time
	mutex       sync.Mutex
}

// windowEntry represents a sliding window with timestamps.
type windowEntry struct {
	timestamps []time.Time
	lastAccess time.Time
	mutex      sync.Mutex
}

// MemoryStore is a secure in-memory implementation of Store
// with automatic expiration and max keys limit.
type MemoryStore struct {
	algorithm Algorithm
	window    time.Duration
	rate      int
	maxKeys   int

	buckets  map[string]*bucketEntry
	counters map[string]*counterEntry
	windows  map[string]*windowEntry

	mu sync.RWMutex
}

// NewMemoryStore creates a new in-memory rate limit store.
// If maxKeys is 0, a default of 10000 is used.
func NewMemoryStore(algorithm Algorithm, window time.Duration, rate, maxKeys int) *MemoryStore {
	if maxKeys <= 0 {
		maxKeys = 10000
	}

	return &MemoryStore{
		algorithm: algorithm,
		window:    window,
		rate:      rate,
		maxKeys:   maxKeys,
		buckets:   make(map[string]*bucketEntry),
		counters:  make(map[string]*counterEntry),
		windows:   make(map[string]*windowEntry),
	}
}

// CheckAndRecord implements Store.
func (s *MemoryStore) CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	switch s.algorithm {
	case TokenBucket:
		return s.checkTokenBucket(key, now)
	case FixedWindow:
		return s.checkFixedWindow(key, now)
	case SlidingWindow:
		return s.checkSlidingWindow(key, now)
	default:
		return s.checkTokenBucket(key, now)
	}
}

func (s *MemoryStore) checkTokenBucket(key string, now time.Time) (bool, int, time.Time) {
	s.mu.Lock()

	entry, exists := s.buckets[key]
	if !exists || now.Sub(entry.lastAccess) > s.window {
		// Entry doesn't exist or expired - create new
		if exists {
			delete(s.buckets, key)
		}
		if len(s.buckets) >= s.maxKeys {
			s.evictOldestBucket()
		}
		entry = &bucketEntry{
			tokens:     float64(s.rate),
			capacity:   float64(s.rate),
			rate:       float64(s.rate) / s.window.Seconds(),
			lastRefill: now,
			lastAccess: now,
		}
		s.buckets[key] = entry
	} else {
		entry.lastAccess = now
	}

	// Release store lock before acquiring entry lock to maintain consistent
	// lock ordering and prevent potential deadlocks
	s.mu.Unlock()

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	elapsed := now.Sub(entry.lastRefill).Seconds()
	entry.tokens = min(entry.capacity, entry.tokens+elapsed*entry.rate)
	entry.lastRefill = now

	resetTime := now.Add(time.Duration((entry.capacity-entry.tokens)/entry.rate) * time.Second)

	if entry.tokens >= 1.0 {
		entry.tokens--
		return true, int(entry.tokens), resetTime
	}

	return false, 0, resetTime
}

func (s *MemoryStore) checkFixedWindow(key string, now time.Time) (bool, int, time.Time) {
	s.mu.Lock()

	entry, exists := s.counters[key]
	if !exists || now.Sub(entry.windowStart) >= s.window {
		// Window expired or new entry
		if exists {
			delete(s.counters, key)
		}
		if len(s.counters) >= s.maxKeys {
			s.evictOldestCounter()
		}
		entry = &counterEntry{
			count:       1,
			windowStart: now,
			lastAccess:  now,
		}
		s.counters[key] = entry
		s.mu.Unlock()
		return true, s.rate - 1, now.Add(s.window)
	}

	entry.lastAccess = now

	// Release store lock before acquiring entry lock to maintain consistent
	// lock ordering and prevent potential deadlocks
	s.mu.Unlock()

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	if entry.count < s.rate {
		entry.count++
		return true, s.rate - entry.count, entry.windowStart.Add(s.window)
	}

	return false, 0, entry.windowStart.Add(s.window)
}

func (s *MemoryStore) checkSlidingWindow(key string, now time.Time) (bool, int, time.Time) {
	s.mu.Lock()

	entry, exists := s.windows[key]
	if !exists || now.Sub(entry.lastAccess) > s.window {
		// Entry expired or new
		if exists {
			delete(s.windows, key)
		}
		if len(s.windows) >= s.maxKeys {
			s.evictOldestWindow()
		}
		entry = &windowEntry{
			timestamps: []time.Time{now},
			lastAccess: now,
		}
		s.windows[key] = entry
		s.mu.Unlock()
		return true, s.rate - 1, now.Add(s.window)
	}

	entry.lastAccess = now

	// Release store lock before acquiring entry lock to maintain consistent
	// lock ordering and prevent potential deadlocks
	s.mu.Unlock()

	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// Remove expired timestamps
	cutoff := now.Add(-s.window)
	newTimestamps := entry.timestamps[:0]
	for _, t := range entry.timestamps {
		if t.After(cutoff) {
			newTimestamps = append(newTimestamps, t)
		}
	}

	if len(newTimestamps) < s.rate {
		newTimestamps = append(newTimestamps, now)
		entry.timestamps = newTimestamps
		remaining := s.rate - len(newTimestamps)
		resetTime := now.Add(s.window)
		if len(newTimestamps) > 0 {
			resetTime = newTimestamps[0].Add(s.window)
		}
		return true, remaining, resetTime
	}

	entry.timestamps = newTimestamps
	resetTime := newTimestamps[0].Add(s.window)
	return false, 0, resetTime
}

// entryWithLastAccess is an interface for entries that have a lastAccess field.
type entryWithLastAccess interface {
	getLastAccess() time.Time
}

func (e *bucketEntry) getLastAccess() time.Time  { return e.lastAccess }
func (e *counterEntry) getLastAccess() time.Time { return e.lastAccess }
func (e *windowEntry) getLastAccess() time.Time  { return e.lastAccess }

// evictOldest removes the entry with the oldest lastAccess time from the map.
// If multiple entries have the same lastAccess, the lexicographically smaller key is chosen.
func evictOldest[M ~map[string]E, E entryWithLastAccess](m M) {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range m {
		accessTime := entry.getLastAccess()
		if first || accessTime.Before(oldestTime) || (accessTime.Equal(oldestTime) && key < oldestKey) {
			oldestKey = key
			oldestTime = accessTime
			first = false
		}
	}

	if oldestKey != "" {
		delete(m, oldestKey)
	}
}

func (s *MemoryStore) evictOldestBucket()  { evictOldest(s.buckets) }
func (s *MemoryStore) evictOldestCounter() { evictOldest(s.counters) }
func (s *MemoryStore) evictOldestWindow()  { evictOldest(s.windows) }

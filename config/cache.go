package config

import (
	"context"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// CacheStore is the interface for cache storage backends.
// Users can implement this interface to provide their own storage
// (e.g., Redis, database, or distributed cache).
type CacheStore interface {
	// Get retrieves a cached response by key.
	// Returns the cached record, true if found, and any error.
	// If not found, returns false and nil error.
	// If an error occurs (e.g., network error), returns false and the error.
	Get(ctx context.Context, key string) (CacheRecord, bool, error)

	// Set stores a response in the cache with the given TTL.
	// Returns an error if the operation fails (e.g., network error for external stores).
	Set(ctx context.Context, key string, record CacheRecord, ttl time.Duration) error
}

// CacheRecord represents a cached HTTP response.
type CacheRecord struct {
	StatusCode   int
	Headers      map[string][]string
	Body         []byte
	ETag         string
	LastModified time.Time
	VaryHeaders  map[string]string
}

// CacheConfig configures the HTTP cache middleware.
type CacheConfig struct {
	// CacheControl sets the Cache-Control header on responses.
	// Default: "private, max-age=60"
	CacheControl string

	// DefaultTTL is the default cache duration.
	// Default: 1m
	DefaultTTL time.Duration

	// MaxBodySize is the maximum response body size to cache (in bytes).
	// Responses larger than this are not cached.
	// Default: 10MB
	MaxBodySize int64

	// MaxEntries is the maximum number of entries to keep in the in-memory cache.
	// Set to 0 for unlimited (not recommended for production).
	// Default: 10000
	MaxEntries int

	// ETag enables automatic ETag generation (SHA256 hash of body).
	// Default: true
	ETag bool

	// LastModified enables automatic Last-Modified timestamp.
	// Default: true
	LastModified bool

	// Vary headers that should be included in the cache key.
	// Default: ["Accept", "Accept-Encoding", "Accept-Language"]
	Vary []string

	// Store is a custom cache store implementation.
	// If nil, an in-memory LRU cache is used.
	// Default: nil
	Store CacheStore

	// ExemptPaths are paths that should not be cached.
	// Default: []
	ExemptPaths []string

	// StatusCodes is a list of status codes that can be cached.
	// Default: [200, 201, 204, 301, 302, 304, 307, 308]
	StatusCodes []int

	// CacheStatusHeader adds a header to responses indicating cache hit/miss.
	// Set to empty string to disable. Default: "X-Cache"
	CacheStatusHeader *string
}

// DefaultCacheConfig is the default configuration for the cache middleware.
var DefaultCacheConfig = CacheConfig{
	CacheControl: "private, max-age=60",
	DefaultTTL:   time.Minute,
	MaxBodySize:  10 * 1024 * 1024,
	MaxEntries:   10000,
	ETag:         true,
	LastModified: true,
	Vary:         []string{httpx.HeaderAccept, httpx.HeaderAcceptEncoding, httpx.HeaderAcceptLanguage},
	ExemptPaths:  []string{},
	StatusCodes:  []int{200, 201, 204, 301, 302, 304, 307, 308},
}

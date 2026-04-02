package cache

import (
	"context"
	"time"

	"github.com/alexferl/zerohttp/httpx"
)

// Store is the interface for cache storage backends.
// Users can implement this interface to provide their own storage
// (e.g., Redis, database, or distributed cache).
// Implement this if you only need cache storage.
// For a backend shared with other middlewares, implement storage.Storage
// and use cache.NewStorageAdapter instead.
type Store interface {
	// Get retrieves a cached response by key.
	// Returns the cached record, true if found, and any error.
	// If not found, returns false and nil error.
	// If an error occurs (e.g., network error), returns false and the error.
	Get(ctx context.Context, key string) (Record, bool, error)

	// Set stores a response in the cache with the given TTL.
	// Returns an error if the operation fails (e.g., network error for external stores).
	Set(ctx context.Context, key string, record Record, ttl time.Duration) error

	// Delete removes a cached response by key.
	// Returns an error if the operation fails.
	Delete(ctx context.Context, key string) error

	// Close releases resources associated with the store.
	// Returns an error if the close operation fails.
	Close() error
}

// Record represents a cached HTTP response.
type Record struct {
	StatusCode   int
	Headers      map[string][]string
	Body         []byte
	ETag         string
	LastModified time.Time
	VaryHeaders  map[string]string
}

// Config configures the HTTP cache middleware.
type Config struct {
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
	Store Store

	// ExcludedPaths are paths that should not be cached.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where caching is explicitly applied.
	// If set, caching will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, caching applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// StatusCodes is a list of status codes that can be cached.
	// Default: [200, 201, 204, 301, 302, 304, 307, 308]
	StatusCodes []int

	// CacheStatusHeader adds a header to responses indicating cache hit/miss.
	// Set to empty string to disable.
	// Default: "X-Cache"
	CacheStatusHeader *string
}

// DefaultConfig is the default configuration for the cache middleware.
var DefaultConfig = Config{
	CacheControl:  "private, max-age=60",
	DefaultTTL:    time.Minute,
	MaxBodySize:   10 * 1024 * 1024,
	MaxEntries:    10000,
	ETag:          true,
	LastModified:  true,
	Vary:          []string{httpx.HeaderAccept, httpx.HeaderAcceptEncoding, httpx.HeaderAcceptLanguage},
	ExcludedPaths: []string{},
	IncludedPaths: []string{},
	StatusCodes:   []int{200, 201, 204, 301, 302, 304, 307, 308},
}

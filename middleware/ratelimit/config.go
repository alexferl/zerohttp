package ratelimit

import (
	"context"
	"net/http"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// Store defines the interface for rate limit storage backends.
type Store interface {
	// CheckAndRecord checks if the request is allowed and records the attempt.
	// Returns (allowed, remainingRequests, resetTime).
	CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time)

	// Close releases resources associated with the store.
	// Returns an error if the close operation fails.
	Close() error
}

// Algorithm defines the rate limiting algorithm
type Algorithm string

const (
	// TokenBucket uses token bucket algorithm
	TokenBucket Algorithm = "token_bucket"
	// SlidingWindow uses sliding window algorithm
	SlidingWindow Algorithm = "sliding_window"
	// FixedWindow uses fixed window algorithm
	FixedWindow Algorithm = "fixed_window"
)

// KeyExtractor defines a function to extract rate limit key from request
type KeyExtractor func(*http.Request) string

// Config allows customization of rate limiting behavior
type Config struct {
	// Rate is requests per window.
	// Default: 100
	Rate int

	// Window is the time window duration.
	// Default: 1 minute
	Window time.Duration

	// Algorithm to use.
	// Default: TokenBucket
	Algorithm Algorithm

	// KeyExtractor function to get rate limit key.
	// Default: IP-based
	KeyExtractor KeyExtractor

	// StatusCode to return when rate limited.
	// Default: 429
	StatusCode int

	// Message to return when rate limited.
	// Default: "Rate limit exceeded"
	Message string

	// IncludeHeaders adds rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, etc.)
	// to responses. Use a pointer to distinguish between "not set" and "explicitly false".
	// Default: true
	IncludeHeaders *bool

	// ExcludedPaths contains paths to skip rate limiting.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// Cannot be used with IncludedPaths - setting both will panic.
	// Default: []
	ExcludedPaths []string

	// IncludedPaths contains paths where rate limiting is explicitly applied.
	// If set, rate limiting will only occur for paths matching these patterns.
	// Supports exact matches, prefixes (ending with /), and wildcards (ending with *).
	// If empty, rate limiting applies to all paths (subject to ExcludedPaths).
	// Cannot be used with ExcludedPaths - setting both will panic.
	// Default: []
	IncludedPaths []string

	// Store is the storage backend for rate limiting.
	// If nil, a secure in-memory store is used.
	Store Store

	// MaxKeys limits the number of unique keys stored in the default
	// in-memory store. Set to 0 for unlimited (not recommended).
	// Default: 10000
	MaxKeys int
}

// DefaultConfig contains the default values for rate limit configuration.
// The default KeyExtractor is IP-based (via ratelimit.IPKeyExtractor).
var DefaultConfig = Config{
	Rate:           100,
	Window:         time.Minute,
	Algorithm:      TokenBucket,
	KeyExtractor:   nil, // Uses ratelimit.IPKeyExtractor() by default
	StatusCode:     http.StatusTooManyRequests,
	Message:        "Rate limit exceeded",
	IncludeHeaders: config.Bool(true),
	ExcludedPaths:  []string{},
	IncludedPaths:  []string{},
}

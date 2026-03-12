package config

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

// RateLimitStore defines the interface for rate limit storage backends.
type RateLimitStore interface {
	CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time)
}

// RateLimitAlgorithm defines the rate limiting algorithm
type RateLimitAlgorithm string

const (
	// TokenBucket uses token bucket algorithm
	TokenBucket RateLimitAlgorithm = "token_bucket"
	// SlidingWindow uses sliding window algorithm
	SlidingWindow RateLimitAlgorithm = "sliding_window"
	// FixedWindow uses fixed window algorithm
	FixedWindow RateLimitAlgorithm = "fixed_window"
)

// KeyExtractor defines a function to extract rate limit key from request
type KeyExtractor func(*http.Request) string

// RateLimitConfig allows customization of rate limiting behavior
type RateLimitConfig struct {
	// Rate is requests per window (defaults to 100)
	Rate int

	// Window is the time window duration (defaults to 1 minute)
	Window time.Duration

	// Algorithm to use (defaults to TokenBucket)
	Algorithm RateLimitAlgorithm

	// KeyExtractor function to get rate limit key (defaults to IP-based)
	KeyExtractor KeyExtractor

	// StatusCode to return when rate limited (defaults to 429)
	StatusCode int

	// Message to return when rate limited
	Message string

	// Headers to include in response
	IncludeHeaders bool

	// ExemptPaths contains paths to skip rate limiting
	ExemptPaths []string

	// Store is the storage backend for rate limiting.
	// If nil, a secure in-memory store is used.
	Store RateLimitStore

	// MaxKeys limits the number of unique keys stored in the default
	// in-memory store. Defaults to 10000. Set to 0 for unlimited (not recommended).
	MaxKeys int
}

// DefaultKeyExtractor extracts IP address as the rate limit key.
// It strips the port from RemoteAddr so all connections from the same IP
// share the same rate limit. For X-Forwarded-For, it uses the first IP.
func DefaultKeyExtractor(r *http.Request) string {
	var ip string

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// Use the first one (client IP)
		ip, _, _ = strings.Cut(xff, ",")
		ip = strings.TrimSpace(ip)
	} else {
		ip = r.RemoteAddr
	}

	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}

	// If SplitHostPort fails (no port), return as-is
	return ip
}

// DefaultRateLimitConfig contains the default values for rate limit configuration.
var DefaultRateLimitConfig = RateLimitConfig{
	Rate:           100,
	Window:         time.Minute,
	Algorithm:      TokenBucket,
	KeyExtractor:   DefaultKeyExtractor,
	StatusCode:     http.StatusTooManyRequests,
	Message:        "Rate limit exceeded",
	IncludeHeaders: true,
	ExemptPaths:    []string{},
}

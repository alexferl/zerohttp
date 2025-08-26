package config

import (
	"net/http"
	"time"
)

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

// DefaultKeyExtractor extracts IP address as the rate limit key
func DefaultKeyExtractor(r *http.Request) string {
	// Use X-Forwarded-For if available, otherwise RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

// RateLimitOption configures rate limit middleware.
type RateLimitOption func(*RateLimitConfig)

// WithRateLimitRate sets the number of requests per window.
func WithRateLimitRate(rate int) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.Rate = rate
	}
}

// WithRateLimitWindow sets the time window duration.
func WithRateLimitWindow(window time.Duration) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.Window = window
	}
}

// WithRateLimitAlgorithm sets the rate limiting algorithm.
func WithRateLimitAlgorithm(algorithm RateLimitAlgorithm) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.Algorithm = algorithm
	}
}

// WithRateLimitKeyExtractor sets the function to extract rate limit key from request.
func WithRateLimitKeyExtractor(keyExtractor KeyExtractor) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.KeyExtractor = keyExtractor
	}
}

// WithRateLimitStatusCode sets the status code to return when rate limited.
func WithRateLimitStatusCode(statusCode int) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.StatusCode = statusCode
	}
}

// WithRateLimitMessage sets the message to return when rate limited.
func WithRateLimitMessage(message string) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.Message = message
	}
}

// WithRateLimitIncludeHeaders sets whether to include rate limit headers in response.
func WithRateLimitIncludeHeaders(include bool) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.IncludeHeaders = include
	}
}

// WithRateLimitExemptPaths sets paths to skip rate limiting.
func WithRateLimitExemptPaths(paths []string) RateLimitOption {
	return func(c *RateLimitConfig) {
		c.ExemptPaths = paths
	}
}

package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/alexferl/zerohttp/config"
)

// Bucket represents a token bucket for rate limiting
type Bucket struct {
	tokens     float64
	capacity   float64
	rate       float64
	lastRefill time.Time
	mutex      sync.Mutex
}

// WindowCounter represents a counter for window-based rate limiting
type WindowCounter struct {
	count       int
	windowStart time.Time
	mutex       sync.Mutex
}

// RateLimit creates a rate limiting middleware
func RateLimit(cfg ...config.RateLimitConfig) func(http.Handler) http.Handler {
	c := config.DefaultRateLimitConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if c.Rate <= 0 {
		c.Rate = config.DefaultRateLimitConfig.Rate
	}
	if c.Window <= 0 {
		c.Window = config.DefaultRateLimitConfig.Window
	}
	if c.Algorithm == "" {
		c.Algorithm = config.DefaultRateLimitConfig.Algorithm
	}
	if c.KeyExtractor == nil {
		c.KeyExtractor = config.DefaultRateLimitConfig.KeyExtractor
	}
	if c.StatusCode == 0 {
		c.StatusCode = config.DefaultRateLimitConfig.StatusCode
	}
	if c.Message == "" {
		c.Message = config.DefaultRateLimitConfig.Message
	}

	buckets := make(map[string]*Bucket)
	counters := make(map[string]*WindowCounter)
	slidingWindows := make(map[string][]time.Time)
	mu := sync.RWMutex{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range c.ExemptPaths {
				if r.URL.Path == exemptPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			key := c.KeyExtractor(r)
			now := time.Now()
			allowed := false
			remaining := 0
			resetTime := now.Add(c.Window)

			mu.Lock()
			defer mu.Unlock()

			switch c.Algorithm {
			case config.TokenBucket:
				bucket, exists := buckets[key]
				if !exists {
					bucket = &Bucket{
						tokens:     float64(c.Rate),
						capacity:   float64(c.Rate),
						rate:       float64(c.Rate) / c.Window.Seconds(),
						lastRefill: now,
					}
					buckets[key] = bucket
				}

				bucket.mutex.Lock()
				// Refill tokens based on elapsed time
				elapsed := now.Sub(bucket.lastRefill).Seconds()
				bucket.tokens = min(bucket.capacity, bucket.tokens+elapsed*bucket.rate)
				bucket.lastRefill = now

				if bucket.tokens >= 1.0 {
					bucket.tokens--
					allowed = true
					remaining = int(bucket.tokens)
				} else {
					remaining = 0
				}
				bucket.mutex.Unlock()

			case config.FixedWindow:
				counter, exists := counters[key]
				if !exists || now.Sub(counter.windowStart) >= c.Window {
					counter = &WindowCounter{
						count:       0,
						windowStart: now,
					}
					counters[key] = counter
				}

				counter.mutex.Lock()
				if counter.count < c.Rate {
					counter.count++
					allowed = true
					remaining = c.Rate - counter.count
				} else {
					remaining = 0
				}
				resetTime = counter.windowStart.Add(c.Window)
				counter.mutex.Unlock()

			case config.SlidingWindow:
				window, exists := slidingWindows[key]
				if !exists {
					window = []time.Time{}
				}

				// Remove expired entries
				cutoff := now.Add(-c.Window)
				newWindow := window[:0]
				for _, t := range window {
					if t.After(cutoff) {
						newWindow = append(newWindow, t)
					}
				}

				if len(newWindow) < c.Rate {
					newWindow = append(newWindow, now)
					allowed = true
					remaining = c.Rate - len(newWindow)
				} else {
					remaining = 0
				}

				slidingWindows[key] = newWindow
				if len(newWindow) > 0 {
					resetTime = newWindow[0].Add(c.Window)
				}
			}

			if c.IncludeHeaders {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(c.Rate))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
				w.Header().Set("X-RateLimit-Window", c.Window.String())
			}

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(resetTime).Seconds())))
				w.WriteHeader(c.StatusCode)
				if c.Message != "" {
					if _, err := fmt.Fprint(w, c.Message); err != nil {
						panic(fmt.Errorf("rate limit message write failed: %w", err))
					}
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

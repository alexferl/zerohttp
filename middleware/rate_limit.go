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
func RateLimit(opts ...config.RateLimitOption) func(http.Handler) http.Handler {
	cfg := config.DefaultRateLimitConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.Rate <= 0 {
		cfg.Rate = config.DefaultRateLimitConfig.Rate
	}
	if cfg.Window <= 0 {
		cfg.Window = config.DefaultRateLimitConfig.Window
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = config.DefaultRateLimitConfig.Algorithm
	}
	if cfg.KeyExtractor == nil {
		cfg.KeyExtractor = config.DefaultRateLimitConfig.KeyExtractor
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = config.DefaultRateLimitConfig.StatusCode
	}
	if cfg.Message == "" {
		cfg.Message = config.DefaultRateLimitConfig.Message
	}

	buckets := make(map[string]*Bucket)
	counters := make(map[string]*WindowCounter)
	slidingWindows := make(map[string][]time.Time)
	mu := sync.RWMutex{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, exemptPath := range cfg.ExemptPaths {
				if r.URL.Path == exemptPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			key := cfg.KeyExtractor(r)
			now := time.Now()
			allowed := false
			remaining := 0
			resetTime := now.Add(cfg.Window)

			mu.Lock()
			defer mu.Unlock()

			switch cfg.Algorithm {
			case config.TokenBucket:
				bucket, exists := buckets[key]
				if !exists {
					bucket = &Bucket{
						tokens:     float64(cfg.Rate),
						capacity:   float64(cfg.Rate),
						rate:       float64(cfg.Rate) / cfg.Window.Seconds(),
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
				if !exists || now.Sub(counter.windowStart) >= cfg.Window {
					counter = &WindowCounter{
						count:       0,
						windowStart: now,
					}
					counters[key] = counter
				}

				counter.mutex.Lock()
				if counter.count < cfg.Rate {
					counter.count++
					allowed = true
					remaining = cfg.Rate - counter.count
				} else {
					remaining = 0
				}
				resetTime = counter.windowStart.Add(cfg.Window)
				counter.mutex.Unlock()

			case config.SlidingWindow:
				window, exists := slidingWindows[key]
				if !exists {
					window = []time.Time{}
				}

				// Remove expired entries
				cutoff := now.Add(-cfg.Window)
				newWindow := window[:0]
				for _, t := range window {
					if t.After(cutoff) {
						newWindow = append(newWindow, t)
					}
				}

				if len(newWindow) < cfg.Rate {
					newWindow = append(newWindow, now)
					allowed = true
					remaining = cfg.Rate - len(newWindow)
				} else {
					remaining = 0
				}

				slidingWindows[key] = newWindow
				if len(newWindow) > 0 {
					resetTime = newWindow[0].Add(cfg.Window)
				}
			}

			if cfg.IncludeHeaders {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Rate))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
				w.Header().Set("X-RateLimit-Window", cfg.Window.String())
			}

			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(resetTime).Seconds())))
				w.WriteHeader(cfg.StatusCode)
				if cfg.Message != "" {
					if _, err := fmt.Fprint(w, cfg.Message); err != nil {
						panic(fmt.Errorf("rate limit message write failed: %w", err))
					}
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

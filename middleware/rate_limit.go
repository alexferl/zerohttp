package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

// RateLimit creates a rate limiting middleware.
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

	// Use provided store or create default in-memory store
	var store RateLimitStore
	if c.Store != nil {
		store = c.Store
	} else {
		maxKeys := c.MaxKeys
		if maxKeys == 0 {
			maxKeys = 10000
		}
		store = NewInMemoryStore(c.Algorithm, c.Window, c.Rate, maxKeys)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg := metrics.SafeRegistry(metrics.GetRegistry(r.Context()))

			for _, exemptPath := range c.ExemptPaths {
				if pathMatches(r.URL.Path, exemptPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			key := c.KeyExtractor(r)
			now := time.Now()
			allowed, remaining, resetTime := store.CheckAndRecord(r.Context(), key, now)

			if c.IncludeHeaders {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(c.Rate))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
				w.Header().Set("X-RateLimit-Window", c.Window.String())
			}

			reg.Gauge("ratelimit_remaining", "key").WithLabelValues(key).Set(float64(remaining))

			if !allowed {
				reg.Counter("ratelimit_rejected_total", "key").WithLabelValues(key).Inc()
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(resetTime).Seconds())))
				detail := problem.NewDetail(c.StatusCode, c.Message)
				_ = detail.Render(w)
				return
			}

			reg.Counter("ratelimit_allowed_total", "key").WithLabelValues(key).Inc()
			next.ServeHTTP(w, r)
		})
	}
}

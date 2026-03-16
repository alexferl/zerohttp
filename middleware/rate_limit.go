package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	zconfig "github.com/alexferl/zerohttp/internal/config"
	"github.com/alexferl/zerohttp/internal/problem"
	"github.com/alexferl/zerohttp/metrics"
)

// DefaultMaxKeys is the default maximum number of unique rate limit keys to store in memory.
const DefaultMaxKeys = 10000

// RateLimit creates a rate limiting middleware.
func RateLimit(cfg ...config.RateLimitConfig) func(http.Handler) http.Handler {
	c := config.DefaultRateLimitConfig
	if len(cfg) > 0 {
		zconfig.Merge(&c, cfg[0])
	}

	if c.KeyExtractor == nil {
		c.KeyExtractor = IPKeyExtractor()
	}

	var store RateLimitStore
	if c.Store != nil {
		store = c.Store
	} else {
		maxKeys := c.MaxKeys
		if maxKeys == 0 {
			maxKeys = DefaultMaxKeys
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
				w.Header().Set(httpx.HeaderXRateLimitLimit, strconv.Itoa(c.Rate))
				w.Header().Set(httpx.HeaderXRateLimitRemaining, strconv.Itoa(remaining))
				w.Header().Set(httpx.HeaderXRateLimitReset, strconv.FormatInt(resetTime.Unix(), 10))
				w.Header().Set(httpx.HeaderXRateLimitWindow, c.Window.String())
			}

			reg.Gauge("ratelimit_remaining", "key").WithLabelValues(key).Set(float64(remaining))

			if !allowed {
				reg.Counter("ratelimit_rejected_total", "key").WithLabelValues(key).Inc()
				w.Header().Set(httpx.HeaderRetryAfter, strconv.Itoa(int(time.Until(resetTime).Seconds())))
				detail := problem.NewDetail(c.StatusCode, c.Message)
				_ = detail.RenderAuto(w, r)
				return
			}

			reg.Counter("ratelimit_allowed_total", "key").WithLabelValues(key).Inc()
			next.ServeHTTP(w, r)
		})
	}
}

// KeyExtractor helpers for common rate limiting scenarios.
// These are convenience wrappers around config.KeyExtractor.

// IPKeyExtractor extracts IP address as the rate limit key.
// It strips the port from RemoteAddr so all connections from the same IP
// share the same rate limit. For X-Forwarded-For, it uses the first IP.
//
// This is the default key extractor.
func IPKeyExtractor() config.KeyExtractor {
	return func(r *http.Request) string {
		var ip string

		if xff := r.Header.Get(httpx.HeaderXForwardedFor); xff != "" {
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
}

// HeaderKeyExtractor creates a key extractor that extracts from the specified header.
// Useful for API key-based rate limiting.
//
// Example:
//
//	middleware.RateLimit(config.RateLimitConfig{
//	    KeyExtractor: middleware.HeaderKeyExtractor("X-API-Key"),
//	})
func HeaderKeyExtractor(header string) config.KeyExtractor {
	return func(r *http.Request) string {
		return r.Header.Get(header)
	}
}

// JWTSubjectKeyExtractor returns a key extractor that extracts the JWT subject claim.
// Falls back to empty string if no JWT claims are present.
// Combine with CompositeKeyExtractor for fallback behavior.
//
// Example:
//
//	// Rate limit by JWT subject, fallback to IP
//	middleware.RateLimit(config.RateLimitConfig{
//	    KeyExtractor: middleware.CompositeKeyExtractor(
//	        middleware.JWTSubjectKeyExtractor(),
//	        middleware.IPKeyExtractor(),
//	    ),
//	})
func JWTSubjectKeyExtractor() config.KeyExtractor {
	return func(r *http.Request) string {
		claims := GetJWTClaims(r)
		return claims.Subject()
	}
}

// ContextKeyExtractor creates a key extractor that retrieves a value from context.
// Useful for rate limiting by authenticated user ID.
//
// Example:
//
//	middleware.RateLimit(config.RateLimitConfig{
//	    KeyExtractor: middleware.ContextKeyExtractor("user_id"),
//	})
func ContextKeyExtractor(key string) config.KeyExtractor {
	return func(r *http.Request) string {
		if val := r.Context().Value(key); val != nil {
			if s, ok := val.(string); ok {
				return s
			}
		}
		return ""
	}
}

// CompositeKeyExtractor combines multiple extractors, using the first non-empty result.
//
// Example:
//
//	// Try JWT subject first, then API key header, then IP
//	middleware.RateLimit(config.RateLimitConfig{
//	    KeyExtractor: middleware.CompositeKeyExtractor(
//	        middleware.JWTSubjectKeyExtractor(),
//	        middleware.HeaderKeyExtractor("X-API-Key"),
//	        middleware.IPKeyExtractor(),
//	    ),
//	})
func CompositeKeyExtractor(extractors ...config.KeyExtractor) config.KeyExtractor {
	return func(r *http.Request) string {
		for _, ex := range extractors {
			if key := ex(r); key != "" {
				return key
			}
		}
		return ""
	}
}

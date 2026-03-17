package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements config.RateLimitStore using Redis for distributed
// rate limiting across multiple server instances.
type RedisStore struct {
	client    *redis.Client
	window    time.Duration
	rate      int
	algorithm config.RateLimitAlgorithm
	keyPrefix string
}

// NewRedisStore creates a new Redis-backed rate limit store.
// This allows rate limiting to work across multiple server instances.
func NewRedisStore(client *redis.Client, algorithm config.RateLimitAlgorithm, window time.Duration, rate int) *RedisStore {
	return &RedisStore{
		client:    client,
		window:    window,
		rate:      rate,
		algorithm: algorithm,
		keyPrefix: "ratelimit:",
	}
}

// CheckAndRecord implements config.RateLimitStore using Redis.
// Uses sliding window algorithm with Redis sorted sets.
func (s *RedisStore) CheckAndRecord(ctx context.Context, key string, now time.Time) (bool, int, time.Time) {
	windowStart := now.Add(-s.window)
	redisKey := s.keyPrefix + key

	// Remove old entries outside the window
	s.client.ZRemRangeByScore(ctx, redisKey, "0", fmt.Sprintf("%d", windowStart.UnixMilli()))

	// Count current entries in window
	count, err := s.client.ZCard(ctx, redisKey).Result()
	if err != nil {
		// On error, allow the request (fail open)
		return true, s.rate - 1, now.Add(s.window)
	}

	if int(count) >= s.rate {
		// Rate limit exceeded
		oldest, _ := s.client.ZRangeWithScores(ctx, redisKey, 0, 0).Result()
		resetTime := now.Add(s.window)
		if len(oldest) > 0 {
			resetTime = time.UnixMilli(int64(oldest[0].Score)).Add(s.window)
		}
		return false, 0, resetTime
	}

	// Add current request to the window
	err = s.client.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: now.UnixNano(),
	}).Err()
	if err != nil {
		return true, s.rate - 1, now.Add(s.window)
	}

	// Set expiry on the key to auto-cleanup
	s.client.Expire(ctx, redisKey, s.window)

	remaining := s.rate - int(count) - 1
	resetTime := now.Add(s.window)

	return true, remaining, resetTime
}

func main() {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v\nMake sure Redis is running: docker run -d --name redis -p 6379:6379 redis:7-alpine", err)
	}

	// Create Redis-backed rate limit store
	rate := 10
	window := time.Minute
	store := NewRedisStore(client, config.SlidingWindow, window, rate)

	// Configure the server with Redis store
	app := zh.New()
	app.Use(middleware.RateLimit(config.RateLimitConfig{
		Store:          store,
		Rate:           rate,
		Window:         window,
		IncludeHeaders: true,
	}))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		limit := w.Header().Get("X-RateLimit-Limit")
		remaining := w.Header().Get("X-RateLimit-Remaining")
		reset := w.Header().Get("X-RateLimit-Reset")

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message":   "Hello with Redis rate limiting!",
			"limit":     limit,
			"remaining": remaining,
			"reset":     reset,
		})
	}))

	log.Fatal(app.Start())
}

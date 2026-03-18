package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/alexferl/zerohttp/middleware"
)

// RedisCacheStore implements config.CacheStore using Redis as the backend.
type RedisCacheStore struct {
	client *redis.Client
	prefix string
}

// cacheRecord is a JSON-serializable representation of config.CacheRecord.
type cacheRecord struct {
	StatusCode   int                 `json:"status_code"`
	Headers      map[string][]string `json:"headers"`
	Body         []byte              `json:"body"`
	ETag         string              `json:"etag"`
	LastModified time.Time           `json:"last_modified"`
	VaryHeaders  map[string]string   `json:"vary_headers"`
}

// NewRedisCacheStore creates a new Redis-backed cache store.
func NewRedisCacheStore(client *redis.Client, prefix string) *RedisCacheStore {
	return &RedisCacheStore{
		client: client,
		prefix: prefix,
	}
}

// makeKey creates a Redis key with optional prefix.
func (c *RedisCacheStore) makeKey(key string) string {
	if c.prefix != "" {
		return c.prefix + ":" + key
	}
	return key
}

// Get retrieves a cached response by key from Redis.
func (c *RedisCacheStore) Get(ctx context.Context, key string) (config.CacheRecord, bool, error) {
	data, err := c.client.Get(ctx, c.makeKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return config.CacheRecord{}, false, nil
	}
	if err != nil {
		return config.CacheRecord{}, false, err
	}

	var record cacheRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return config.CacheRecord{}, false, fmt.Errorf("failed to unmarshal cache record: %w", err)
	}

	return config.CacheRecord{
		StatusCode:   record.StatusCode,
		Headers:      record.Headers,
		Body:         record.Body,
		ETag:         record.ETag,
		LastModified: record.LastModified,
		VaryHeaders:  record.VaryHeaders,
	}, true, nil
}

// Set stores a response in Redis with the given TTL.
func (c *RedisCacheStore) Set(ctx context.Context, key string, record config.CacheRecord, ttl time.Duration) error {
	redisRecord := cacheRecord{
		StatusCode:   record.StatusCode,
		Headers:      record.Headers,
		Body:         record.Body,
		ETag:         record.ETag,
		LastModified: record.LastModified,
		VaryHeaders:  record.VaryHeaders,
	}

	data, err := json.Marshal(redisRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal cache record: %w", err)
	}

	return c.client.Set(ctx, c.makeKey(key), data, ttl).Err()
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

	// Create Redis-backed cache store
	cacheStore := NewRedisCacheStore(client, "zerohttp:cache")

	app := zh.New()

	// Public API endpoint - cached for 30 seconds using Redis
	// Demonstrates automatic ETag and Last-Modified generation
	app.GET("/api/public/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		data := map[string]any{
			"timestamp": time.Now().Unix(),
			"message":   "This response is cached in Redis with ETag support",
			"hits":      1,
		}
		return zh.R.JSON(w, http.StatusOK, data)
	}),
		middleware.Cache(config.CacheConfig{
			CacheControl: "public, max-age=30",
			DefaultTTL:   30 * time.Second,
			ETag:         true,
			LastModified: true,
			Store:        cacheStore,
		}),
	)

	// User profile - cached privately per user in Redis
	// Demonstrates Vary header support (different cache per Accept)
	app.GET("/api/users/{id}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		userID := r.PathValue("id")
		data := map[string]any{
			"id":        userID,
			"name":      "User " + userID,
			"email":     fmt.Sprintf("user%s@example.com", userID),
			"fetchedAt": time.Now().Format(time.RFC3339),
		}
		return zh.R.JSON(w, http.StatusOK, data)
	}),
		middleware.Cache(config.CacheConfig{
			CacheControl: "private, max-age=60",
			DefaultTTL:   time.Minute,
			ETag:         true,
			Vary:         []string{httpx.HeaderAccept, httpx.HeaderAcceptEncoding},
			Store:        cacheStore,
		}),
	)

	// Live/health endpoint - never cached
	// Demonstrates exempt paths (or you could just not apply the middleware)
	app.GET("/api/live", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339Nano),
		})
	}),
	// No cache middleware - this endpoint is never cached
	)

	// Static content - aggressively cached in Redis
	// Demonstrates long-term caching with immutable directive
	app.GET("/api/config", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"version":     "1.0.0",
			"features":    []string{"cache", "etag", "ratelimit"},
			"maintenance": false,
		})
	}),
		middleware.Cache(config.CacheConfig{
			CacheControl: "public, max-age=3600, immutable",
			DefaultTTL:   time.Hour,
			Store:        cacheStore,
		}),
	)

	// HTML response - demonstrates text/html caching in Redis
	app.GET("/page/info", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Cache Demo</title></head>
<body>
<h1>Cached HTML Page</h1>
<p>Generated at: %s</p>
<p>This page is cached for 2 minutes.</p>
</body>
</html>`, time.Now().Format(time.RFC3339))
		return zh.R.HTML(w, http.StatusOK, html)
	}),
		middleware.Cache(config.CacheConfig{
			CacheControl: "public, max-age=120",
			DefaultTTL:   2 * time.Minute,
			Store:        cacheStore,
		}),
	)

	// Stats endpoint - demonstrates short-term caching with Redis
	app.GET("/api/stats", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]any{
			"requests":    12345,
			"cacheHits":   9876,
			"cacheMisses": 2469,
			"updatedAt":   time.Now().Unix(),
		})
	}),
		middleware.Cache(config.CacheConfig{
			CacheControl: "public, max-age=10",
			DefaultTTL:   10 * time.Second,
			Store:        cacheStore,
		}),
	)

	log.Fatal(app.Start())
}

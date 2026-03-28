package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/redis/go-redis/v9"
)

// RedisIdempotencyStore implements idempotency.Store using Redis as the backend.
type RedisIdempotencyStore struct {
	client  *redis.Client
	prefix  string
	lockTTL time.Duration
}

// recordData is a JSON-serializable representation of idempotency.Record.
type recordData struct {
	StatusCode int      `json:"status_code"`
	Headers    []string `json:"headers"`
	Body       []byte   `json:"body"`
	CreatedAt  int64    `json:"created_at"`
}

// NewRedisIdempotencyStore creates a new Redis-backed idempotency store.
// lockTTL specifies how long to hold the distributed lock.
// Use 0 for no expiry, negative value for default (30s).
func NewRedisIdempotencyStore(client *redis.Client, prefix string, lockTTL time.Duration) *RedisIdempotencyStore {
	if lockTTL < 0 {
		lockTTL = 30 * time.Second
	}
	return &RedisIdempotencyStore{
		client:  client,
		prefix:  prefix,
		lockTTL: lockTTL,
	}
}

// makeKey creates a Redis key with optional prefix.
func (s *RedisIdempotencyStore) makeKey(key string) string {
	if s.prefix != "" {
		return s.prefix + ":" + key
	}
	return key
}

// Get retrieves a cached response by key from Redis.
func (s *RedisIdempotencyStore) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	data, err := s.client.Get(ctx, s.makeKey(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return idempotency.Record{}, false, nil
	}
	if err != nil {
		return idempotency.Record{}, false, err
	}

	var rd recordData
	if err := json.Unmarshal(data, &rd); err != nil {
		return idempotency.Record{}, false, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return idempotency.Record{
		StatusCode: rd.StatusCode,
		Headers:    rd.Headers,
		Body:       rd.Body,
		CreatedAt:  time.Unix(rd.CreatedAt, 0),
	}, true, nil
}

// Set stores a response in Redis with the given TTL.
func (s *RedisIdempotencyStore) Set(ctx context.Context, key string, record idempotency.Record, ttl time.Duration) error {
	rd := recordData{
		StatusCode: record.StatusCode,
		Headers:    record.Headers,
		Body:       record.Body,
		CreatedAt:  record.CreatedAt.Unix(),
	}

	data, err := json.Marshal(rd)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	return s.client.Set(ctx, s.makeKey(key), data, ttl).Err()
}

// Lock attempts to acquire a distributed lock for the given key.
// Returns true if lock was acquired, false if already locked.
func (s *RedisIdempotencyStore) Lock(ctx context.Context, key string) (bool, error) {
	lockKey := s.makeKey(key) + ":lock"
	ok, err := s.client.SetArgs(ctx, lockKey, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  s.lockTTL,
	}).Result()
	if err != nil {
		return false, err
	}
	return ok != "", nil
}

// Unlock releases the distributed lock for the given key.
func (s *RedisIdempotencyStore) Unlock(ctx context.Context, key string) error {
	lockKey := s.makeKey(key) + ":lock"
	return s.client.Del(ctx, lockKey).Err()
}

// Close releases resources associated with the store.
func (s *RedisIdempotencyStore) Close() error {
	return s.client.Close()
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

	// Create Redis-backed idempotency store
	store := NewRedisIdempotencyStore(client, "zerohttp:idempotency", 30*time.Second)

	app := zh.New()

	// Apply idempotency middleware to payment endpoint
	// This ensures duplicate requests with the same Idempotency-Key
	// return the same response without re-processing
	app.POST("/api/payments", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Amount   float64 `json:"amount"`
			Currency string  `json:"currency"`
			To       string  `json:"to"`
		}

		if err := zh.B.JSON(r.Body, &req); err != nil {
			return zh.R.JSON(w, http.StatusBadRequest, zh.M{"error": "invalid request"})
		}

		// Simulate payment processing
		time.Sleep(100 * time.Millisecond)

		// Return success response
		return zh.R.JSON(w, http.StatusCreated, zh.M{
			"id":       fmt.Sprintf("pay_%d", time.Now().Unix()),
			"amount":   req.Amount,
			"currency": req.Currency,
			"to":       req.To,
			"status":   "completed",
		})
	}),
		idempotency.New(idempotency.Config{
			Store:      store,
			TTL:        time.Hour,
			HeaderName: "Idempotency-Key",
		}),
	)

	// Regular endpoint without idempotency (for comparison)
	app.POST("/api/regular", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message":   "This endpoint is not idempotent",
			"timestamp": time.Now().Unix(),
		})
	}))

	log.Fatal(app.Start())
}

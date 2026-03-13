# Idempotency Example

This example demonstrates the idempotency middleware for handling duplicate requests safely.

## Features Demonstrated

- **Request deduplication** - Cache responses and replay for identical requests
- **Composite cache keys** - Key + Method + Path + Body for uniqueness
- **In-flight request handling** - Concurrent requests with same key wait for completion
- **Configurable TTL** - Automatic expiration of cached responses
- **Required vs optional** - Enforce idempotency keys for critical operations
- **Exempt paths** - Skip idempotency for specific endpoints
- **Size limits** - Skip caching for large request bodies

## How It Works

When a state-changing request (POST, PUT, PATCH, DELETE) includes an `Idempotency-Key` header:

1. **First request** - Process normally, cache the response
2. **Duplicate request** - Return cached response with `X-Idempotency-Replay: true` header
3. **In-flight request** - Wait for completion (with retry/backoff)
4. **Max retries exceeded** - Return `409 Conflict` if request still processing

The cache key is a composite of: `idempotencyKey:Method:Path:Body`

## Running the Example

```bash
go run main.go
```

## Test Commands

### 1. Create Payment (First Request - Cached)
```bash
curl -i -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: key-123' \
  -d '{"amount":100}'
```

Returns `201 Created` with a new payment ID.

### 2. Same Payment with Same Key (Cached Response)
```bash
curl -i -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: key-123' \
  -d '{"amount":100}'
```

Returns the **same payment ID** with `X-Idempotency-Replay: true` header.

### 3. Different Body with Same Key (New Request)
```bash
curl -i -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: key-123' \
  -d '{"amount":200}'
```

Creates a **new payment** because the body differs (different cache key).

### 4. Required Idempotency Key (Fails Without Key)
```bash
curl -i -X POST http://localhost:8080/api/transfers \
  -H 'Content-Type: application/json' \
  -d '{"amount":500}'
```

Returns `400 Bad Request` - idempotency key is required for transfers.

### 5. Required Idempotency Key (Succeeds With Key)
```bash
curl -i -X POST http://localhost:8080/api/transfers \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: transfer-456' \
  -d '{"amount":500}'
```

Returns `201 Created` and caches the response.

### 6. Exempt Path (Webhook - No Idempotency)
```bash
curl -i -X POST http://localhost:8080/api/webhooks \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: webhook-789' \
  -d '{"event":"payment.received"}'
```

Processes the webhook normally (no caching), even with an idempotency key.

### 7. Large Body (Exceeds MaxBodySize)
```bash
curl -i -X POST http://localhost:8080/api/bulk-import \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: bulk-001' \
  -d '{"data":"<2000+ characters>"}'
```

Request body exceeds 1KB limit, so the request is **not cached**.

## Configuration Options

```go
middleware.Idempotency(config.IdempotencyConfig{
    HeaderName:            "Idempotency-Key",    // Header to read key from
    TTL:                   24 * time.Hour,       // Cache duration
    MaxBodySize:           1024 * 1024,          // Max body size to cache (bytes)
    Required:              false,                // Require key for state-changing methods
    ExemptPaths:           []string{},           // Paths to skip
    MaxKeys:               10000,                // In-memory store limit
    LockRetryInterval:     10 * time.Millisecond,// Initial retry interval
    LockMaxRetries:        300,                  // Max retries for in-flight
    LockMaxInterval:       500 * time.Millisecond,// Max retry interval
    LockBackoffMultiplier: 2.0,                  // Exponential backoff multiplier
})
```

## Production Considerations

For multi-instance deployments, use a shared store like Redis:

```go
type RedisIdempotencyStore struct {
    client *redis.Client
}

func (s *RedisIdempotencyStore) Get(ctx context.Context, key string) (config.IdempotencyRecord, bool, error) {
    data, err := s.client.Get(ctx, "idempotency:"+key).Bytes()
    if err == redis.Nil {
        return config.IdempotencyRecord{}, false, nil
    }
    if err != nil {
        return config.IdempotencyRecord{}, false, err
    }
    var record config.IdempotencyRecord
    if err := json.Unmarshal(data, &record); err != nil {
        return config.IdempotencyRecord{}, false, err
    }
    return record, true, nil
}

func (s *RedisIdempotencyStore) Set(ctx context.Context, key string, record config.IdempotencyRecord, ttl time.Duration) error {
    data, err := json.Marshal(record)
    if err != nil {
        return err
    }
    return s.client.Set(ctx, "idempotency:"+key, data, ttl).Err()
}

func (s *RedisIdempotencyStore) Lock(ctx context.Context, key string) (bool, error) {
    // Use Redis SETNX for distributed locking
    acquired, err := s.client.SetNX(ctx, "idempotency:lock:"+key, "1", 30*time.Second).Result()
    return acquired, err
}

func (s *RedisIdempotencyStore) Unlock(ctx context.Context, key string) error {
    return s.client.Del(ctx, "idempotency:lock:"+key).Err()
}

// Usage
app.Use(middleware.Idempotency(config.IdempotencyConfig{
    Store: &RedisIdempotencyStore{client: redisClient},
}))
```

## Key Behavior Notes

1. **Only state-changing methods** - GET/HEAD/OPTIONS are never cached
2. **2xx responses only** - Error responses are not cached
3. **Body matters** - Same key with different body = different request
4. **Fail open** - Store errors don't block requests (logged only)
5. **Flat header storage** - Headers stored as `[key1, val1, key2, val2...]` for efficiency

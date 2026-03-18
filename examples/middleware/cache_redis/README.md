# HTTP Cache Middleware with Redis Example

This example demonstrates zerohttp's HTTP cache middleware using Redis as the cache store backend.

## Features

- Redis-backed distributed caching
- Public/private cache control directives
- ETag generation for conditional requests (304 Not Modified)
- Last-Modified header support
- Vary header support for content negotiation
- Per-route cache configuration with custom store

## Prerequisites

### Start Redis with Docker

```bash
# Start Redis container
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Verify Redis is running
docker ps

# View Redis logs
docker logs redis
```

### Or use Docker Compose

```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

```bash
docker-compose up -d
```

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint               | Cache Policy              | Description                      |
|------------------------|---------------------------|----------------------------------|
| `GET /api/public/data` | Public, 30s TTL           | Basic cached response with ETag  |
| `GET /api/users/{id}`  | Private, 60s TTL          | Per-user cache with Vary headers |
| `GET /api/live`        | No cache                  | Live timestamp (never cached)    |
| `GET /api/config`      | Public, 1h TTL, immutable | Aggressively cached config       |
| `GET /page/info`       | Public, 2m TTL            | Cached HTML page                 |
| `GET /api/stats`       | Public, 10s TTL           | Short-term cached stats          |

## Test Commands

### 1. Basic cached request (30s TTL)
```bash
curl -i http://localhost:8080/api/public/data
```

Make the same request again within 30 seconds - notice the timestamp stays the same (served from Redis cache).

### 2. Conditional request with ETag
```bash
# First, get the ETag from the response above, then:
curl -i http://localhost:8080/api/public/data -H 'If-None-Match: "<etag-from-above>"'
# Returns 304 Not Modified if content hasn't changed
```

### 3. User profile (private cache)
```bash
curl -i http://localhost:8080/api/users/123
```

### 4. Live endpoint (never cached)
```bash
curl -i http://localhost:8080/api/live
curl -i http://localhost:8080/api/live
# Notice the timestamps are always different
```

### 5. Static config (1h TTL)
```bash
curl -i http://localhost:8080/api/config
```

### 6. HTML page (2m TTL)
```bash
curl -i http://localhost:8080/page/info
```

### 7. Stats endpoint (10s TTL)
```bash
curl -i http://localhost:8080/api/stats
```

### 8. Inspect Redis cache directly
```bash
# List all cache keys
redis-cli keys 'zerohttp:cache:*'

# Get a specific cache entry
redis-cli get 'zerohttp:cache:GET|/api/public/data|'
```

## Implementation Details

The `RedisCacheStore` implements the `config.CacheStore` interface:

```go
type CacheStore interface {
    Get(ctx context.Context, key string) (CacheRecord, bool, error)
    Set(ctx context.Context, key string, record CacheRecord, ttl time.Duration) error
}
```

Cache records are JSON-serialized before storage in Redis.

## Cleanup

```bash
# Stop and remove Redis container
docker stop redis
docker rm redis
```

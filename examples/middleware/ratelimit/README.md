# Rate Limit Example

This example demonstrates zerohttp's rate limiting middleware with various configurations.

## Features

- Token bucket and sliding window algorithms
- Per-client rate limiting with custom key extractors
- Rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)
- Path exclusions for health checks
- Configurable in-memory store

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint               | Rate Limit         | Description                     |
|------------------------|--------------------|---------------------------------|
| `GET /api/default`     | 100 req/min        | Default token bucket limiter    |
| `GET /api/strict`      | 10 req/second      | Strict sliding window limiter   |
| `GET /api/user`        | 5 req/min per user | Per-user rate limiting          |
| `GET /api/high-volume` | 1000 req/min       | High-volume endpoint            |
| `GET /health`          | Excluded           | Health check (no rate limit)    |

## Test Commands

### Default rate limit (100 req/min)
```bash
curl -i http://localhost:8080/api/default
```

### Strict rate limit (10 req/second)
```bash
for i in {1..15}; do curl -s http://localhost:8080/api/strict; echo; done
```

### Per-user rate limit with custom header
```bash
# User "alice" - 5 requests per minute
curl -H "X-User-ID: alice" http://localhost:8080/api/user

# User "bob" - separate limit bucket
curl -H "X-User-ID: bob" http://localhost:8080/api/user

# Anonymous (no header) - shared bucket
curl http://localhost:8080/api/user
```

### Check rate limit headers
```bash
curl -i http://localhost:8080/api/default
```

Response includes:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704067200
```

### Health check (excluded from rate limiting)
```bash
curl -i http://localhost:8080/health
```

## Rate Limit Response

When the rate limit is exceeded, the server returns:
```json
{
  "error": "rate limit exceeded",
  "retry_after": 45
}
```

With HTTP status `429 Too Many Requests` and header:
```
Retry-After: 45
```

## Algorithms

### Token Bucket (default)
- Allows short bursts up to the bucket capacity
- Smooths traffic over time
- Good for general API rate limiting

### Sliding Window
- More strict than token bucket
- Counts requests in a rolling time window
- Better for preventing abuse

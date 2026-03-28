# Idempotency Middleware with Redis Example

This example demonstrates idempotent request handling using Redis as the storage backend. This ensures safe retries for state-changing operations across multiple server instances.

## Features

- Redis-backed idempotency storage
- Distributed locking to prevent concurrent request processing
- Automatic deduplication of requests with the same idempotency key
- Configurable TTL for cached responses
- Per-route idempotency configuration

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

| Endpoint            | Idempotency | Description                          |
|---------------------|-------------|--------------------------------------|
| `POST /api/payments`| Yes         | Payment processing with idempotency  |
| `POST /api/regular` | No          | Regular endpoint (for comparison)    |

## Test Commands

### 1. Idempotent payment request

```bash
curl -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: payment-123' \
  -d '{"amount":100.00,"currency":"USD","to":"merchant123"}'
```

Response:
```json
{
  "id": "pay_1704067200",
  "amount": 100.00,
  "currency": "USD",
  "to": "merchant123",
  "status": "completed"
}
```

### 2. Send the same request again (deduplication)

```bash
curl -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: payment-123' \
  -d '{"amount":100.00,"currency":"USD","to":"merchant123"}'
```

Notice the response is identical (same `id`) - the request was deduplicated and the cached response was returned without re-processing the payment.

### 3. Different idempotency key (new request)

```bash
curl -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: payment-456' \
  -d '{"amount":200.00,"currency":"USD","to":"merchant456"}'
```

This creates a new payment with a different `id`.

### 4. Regular endpoint (no idempotency)

```bash
curl -X POST http://localhost:8080/api/regular
```

Each request returns a different timestamp (not idempotent).

### 5. Inspect Redis directly

```bash
# List all idempotency keys
redis-cli keys 'zerohttp:idempotency:*'

# Get a specific cached response
redis-cli get 'zerohttp:idempotency:payment-123'

# Check for active locks (should be empty unless processing)
redis-cli keys 'zerohttp:idempotency:*:lock'
```

## Implementation Details

The `RedisIdempotencyStore` implements the `idempotency.Store` interface:

```go
type Store interface {
    Get(ctx context.Context, key string) (Record, bool, error)
    Set(ctx context.Context, key string, record Record, ttl time.Duration) error
    Lock(ctx context.Context, key string) (bool, error)
    Unlock(ctx context.Context, key string) error
    Close() error
}
```

- **Get**: Retrieves cached responses from Redis
- **Set**: Stores responses with TTL for automatic expiration
- **Lock/Unlock**: Distributed locking using Redis SETNX to prevent concurrent processing
- Records are JSON-serialized before storage

## Cleanup

```bash
# Stop and remove Redis container
docker stop redis
docker rm redis
```

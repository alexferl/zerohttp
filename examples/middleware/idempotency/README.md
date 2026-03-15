# Idempotency Example

This example demonstrates zerohttp's idempotency middleware for safe request retries.

## Features

- In-memory response caching by idempotency key
- Required vs optional idempotency keys
- Exempt paths for webhooks
- Body size limits for caching
- Idempotency-Replay header for cached responses

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint                | Idempotency | Description                      |
|-------------------------|-------------|----------------------------------|
| `POST /api/payments`    | Optional    | Basic idempotency with caching   |
| `POST /api/transfers`   | Required    | Fails without idempotency key    |
| `POST /api/webhooks`    | Exempt      | Skips idempotency check          |
| `POST /api/bulk-import` | Optional    | Max body size limit (1KB)        |
| `GET /api/status`       | N/A         | GET requests bypass idempotency  |

## Test Commands

### Create payment with idempotency key (first request - cached)
```bash
curl -i -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: key-123' \
  -d '{"amount":100}'
```

### Same payment with same key (returns cached response)
```bash
curl -i -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: key-123' \
  -d '{"amount":100}'
```
Notice: Same payment ID returned, `X-Idempotency-Replay: true` header

### Different body with same key (not replayed - body differs)
```bash
curl -i -X POST http://localhost:8080/api/payments \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: key-123' \
  -d '{"amount":200}'
```

### Required idempotency key (fails without key)
```bash
curl -i -X POST http://localhost:8080/api/transfers \
  -H 'Content-Type: application/json' \
  -d '{"amount":500}'
```

### Required idempotency key (succeeds with key)
```bash
curl -i -X POST http://localhost:8080/api/transfers \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: transfer-456' \
  -d '{"amount":500}'
```

### Exempt path (webhook - no idempotency check)
```bash
curl -i -X POST http://localhost:8080/api/webhooks \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: webhook-789' \
  -d '{"event":"payment.received"}'
```

### Large body (exceeds MaxBodySize, not cached)
```bash
curl -i -X POST http://localhost:8080/api/bulk-import \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: bulk-001' \
  -d '{"data":"'$(head -c 2000 < /dev/zero | tr '\0' 'a')'"}'
```

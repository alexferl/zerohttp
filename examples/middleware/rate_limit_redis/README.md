# Rate Limit with Redis Example

This example demonstrates distributed rate limiting using Redis as the backing store. This allows rate limiting to work consistently across multiple server instances.

## Features

- Distributed rate limiting with Redis
- Sliding window algorithm
- Shared rate limit state across server instances
- Automatic key expiration and cleanup

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

| Endpoint | Rate Limit   | Description                     |
|----------|--------------|---------------------------------|
| `GET /`  | 10 req/min   | Hello endpoint with rate limits |

## Test Commands

### Make requests until rate limited
```bash
for i in {1..12}; do curl -s http://localhost:8080/; echo; done
```

After 10 requests, you will see:
```json
{"error":"rate limit exceeded","retry_after":60}
```

### Check rate limit headers
```bash
curl -i http://localhost:8080/
```

Response includes:
```
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 9
X-RateLimit-Reset: 1704067200
```

### Simulate multiple instances

Run the example in two separate terminals. Both will share the same rate limit counter in Redis:

```bash
# Terminal 1
go run .

# Terminal 2 (in another window)
go run .
```

Make requests to either instance - they both count toward the same limit.

## Cleanup

```bash
# Stop and remove Redis container
docker stop redis
docker rm redis
```

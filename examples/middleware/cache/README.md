# HTTP Cache Middleware Example

This example demonstrates zerohttp's HTTP cache middleware with various caching strategies.

## Features

- Public/private cache control directives
- ETag generation for conditional requests (304 Not Modified)
- Last-Modified header support
- Vary header support for content negotiation
- Per-route cache configuration

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

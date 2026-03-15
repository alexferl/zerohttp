# No-Cache Middleware Example

This example demonstrates zerohttp's no-cache middleware for preventing HTTP caching on sensitive or dynamic endpoints.

## Features

- Global no-cache middleware application
- Per-route custom no-cache configuration
- Automatic removal of conditional request headers (ETag, If-Modified-Since, etc.)
- Proper Cache-Control, Pragma, and Expires headers

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint                 | Description                           |
|--------------------------|---------------------------------------|
| `GET /api/user/profile`  | Sensitive user data (never cached)    |
| `GET /api/session`       | Session information (never cached)    |
| `GET /api/live/status`   | Live system status (never cached)     |
| `GET /api/admin/secrets` | Custom no-cache headers configuration |

## Test Commands

### Check no-cache headers on a response
```bash
curl -i http://localhost:8080/api/user/profile
```

Notice the response headers:
```
Cache-Control: no-cache, no-store, no-transform, must-revalidate, private, max-age=0
Pragma: no-cache
Expires: Thu, 01 Jan 1970 00:00:00 GMT
X-Accel-Expires: 0
```

### Verify timestamp changes on every request (not cached)
```bash
for i in {1..5}; do curl -s http://localhost:8080/api/live/status | grep timestamp; done
```

### Test with conditional headers (they get stripped)
```bash
curl -i http://localhost:8080/api/session \
  -H "If-None-Match: \"abc123\"" \
  -H "If-Modified-Since: Wed, 21 Oct 2015 07:28:00 GMT"
```

The middleware removes these headers from the request before it reaches the handler.

### Custom no-cache configuration endpoint
```bash
curl -i http://localhost:8080/api/admin/secrets
```

## Default No-Cache Headers

The middleware sets the following headers by default:

| Header            | Value                                                                    |
|-------------------|--------------------------------------------------------------------------|
| `Cache-Control`   | `no-cache, no-store, no-transform, must-revalidate, private, max-age=0`  |
| `Pragma`          | `no-cache`                                                               |
| `Expires`         | `Thu, 01 Jan 1970 00:00:00 UTC` (Epoch)                                  |
| `X-Accel-Expires` | `0`                                                                      |

## Removed Request Headers

The middleware removes these conditional headers from incoming requests:

- `ETag`
- `If-Modified-Since`
- `If-Match`
- `If-None-Match`
- `If-Range`
- `If-Unmodified-Since`

## When to Use

Use the no-cache middleware for:

- **Sensitive data**: User profiles, financial information, admin panels
- **Dynamic content**: Live status, real-time metrics, system health
- **Authenticated endpoints**: Session data, tokens, user-specific content
- **Form submissions**: POST-redirect-GET patterns

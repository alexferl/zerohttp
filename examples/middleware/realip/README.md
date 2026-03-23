# Real IP Example

This example demonstrates extracting the real client IP when running behind a reverse proxy or load balancer.

## Features

- Automatic extraction from `X-Forwarded-For`, `X-Real-IP`, and `Forwarded` headers
- Custom IP extractors for different proxy setups
- Updates `r.RemoteAddr` with the real client IP

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint       | IP Extraction Method                          |
|----------------|-----------------------------------------------|
| `GET /`        | Default (X-Forwarded-For, X-Real-IP, etc.)    |
| `GET /nginx`   | X-Real-IP only (Nginx style)                  |
| `GET /direct`  | RemoteAddr only (ignores proxy headers)       |

## Test Commands

### Simulate request through a proxy
```bash
curl -i http://localhost:8080/ -H "X-Forwarded-For: 203.0.113.42"
```

### With X-Real-IP header (Nginx style)
```bash
curl -i http://localhost:8080/ -H "X-Real-IP: 198.51.100.10"
```

### Multiple IPs in X-Forwarded-For (first one is used)
```bash
curl -i http://localhost:8080/ -H "X-Forwarded-For: 203.0.113.42, 10.0.0.1, 192.168.1.1"
```

### RFC 7239 Forwarded header
```bash
curl -i http://localhost:8080/ -H "Forwarded: for=203.0.113.42;proto=https"
```

### Direct request (no proxy headers)
```bash
curl -i http://localhost:8080/direct
```

## Supported Headers

The default extractor checks these headers in order:

1. **X-Forwarded-For** - Most common, may contain multiple IPs (first is used)
2. **X-Real-IP** - Used by Nginx and some proxies
3. **X-Forwarded** - Less common legacy header
4. **Forwarded** - RFC 7239 standard header
5. **RemoteAddr** - Fallback to connection address

## Common Proxy Setups

### Nginx
```nginx
location / {
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_pass http://backend;
}
```

### Apache
```apache
ProxyPass / http://backend/
ProxyPassReverse / http://backend/
RequestHeader set X-Forwarded-For %{REMOTE_ADDR}s
```

### Traefik
Traefik automatically sets `X-Forwarded-For` and `X-Real-Ip` headers.

### Cloudflare
Cloudflare sends `CF-Connecting-IP` header. Use a custom extractor:

```go
middleware.RealIP(config.RealIPConfig{
    IPExtractor: func(r *http.Request) string {
        if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
            return cf
        }
        return config.DefaultIPExtractor(r)
    },
})
```

## Security Note

Only use this middleware when your server is behind a trusted proxy. Malicious clients can spoof these headers if requests come directly to your server.

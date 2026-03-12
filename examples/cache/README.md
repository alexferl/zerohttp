# HTTP Cache Example

This example demonstrates the HTTP cache middleware with automatic ETag generation, conditional requests, and Cache-Control handling.

## Features Demonstrated

- **Automatic ETag generation** - SHA256-based content hashes
- **Conditional requests** - 304 Not Modified responses
- **Cache-Control headers** - public, private, max-age directives
- **Vary header support** - Different cache entries per Accept/Accept-Encoding
- **TTL-based expiration** - Automatic cache cleanup
- **Per-route configuration** - Different cache settings per endpoint

## Running the Example

```bash
go run main.go
```

## Test Commands

### 1. Basic Cached Request (30s TTL)
```bash
curl -i http://localhost:8080/api/public/data
```
Notice the `ETag` and `Last-Modified` headers in the response.

### 2. Conditional Request (304 Not Modified)
```bash
# Copy the ETag from the previous response
curl -i http://localhost:8080/api/public/data -H 'If-None-Match: "abc123..."'
```
Returns `304 Not Modified` with no body if content hasn't changed.

### 3. Private Cache (User Profile)
```bash
curl -i http://localhost:8080/api/users/123
curl -i http://localhost:8080/api/users/123 -H 'Accept: application/xml'
```
Each Accept header gets its own cache entry (Vary header support).

### 4. Non-Cached Endpoint (Live/Health)
```bash
curl -i http://localhost:8080/api/live
```
Timestamp changes on every request (no cache middleware applied).

### 5. Long-Term Cache (1 hour TTL)
```bash
curl -i http://localhost:8080/api/config
```
Uses `immutable` directive - browsers won't revalidate during page session.

### 6. HTML Caching
```bash
curl -i http://localhost:8080/page/info
```
HTML content cached with 2-minute TTL.

## Cache Headers Explained

- `Cache-Control: public` - Response may be cached by any cache
- `Cache-Control: private` - Response may only be cached by browser
- `Cache-Control: max-age=N` - Cache for N seconds
- `Cache-Control: immutable` - Content will never change (during page session)
- `ETag` - Content identifier for conditional requests
- `Last-Modified` - Timestamp for conditional requests

## Production Considerations

For multi-instance deployments, use a shared cache store like Redis:

```go
type RedisCacheStore struct {
    client *redis.Client
}

func (s *RedisCacheStore) Get(ctx context.Context, key string) (config.CacheRecord, bool, error) {
    data, err := s.client.Get(ctx, "cache:"+key).Bytes()
    if err == redis.Nil {
        return config.CacheRecord{}, false, nil
    }
    if err != nil {
        return config.CacheRecord{}, false, err
    }
    var record config.CacheRecord
    if err := json.Unmarshal(data, &record); err != nil {
        return config.CacheRecord{}, false, err
    }
    return record, true, nil
}

func (s *RedisCacheStore) Set(ctx context.Context, key string, record config.CacheRecord, ttl time.Duration) error {
    data, err := json.Marshal(record)
    if err != nil {
        return err
    }
    return s.client.Set(ctx, "cache:"+key, data, ttl).Err()
}

// Usage
app.Use(middleware.Cache(config.CacheConfig{
    Store: &RedisCacheStore{client: redisClient},
}))
```

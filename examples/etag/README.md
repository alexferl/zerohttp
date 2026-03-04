# ETag Middleware Example

This example demonstrates the ETag middleware features in zerohttp:

- Automatic ETag generation for dynamic content
- 304 Not Modified responses for conditional requests
- File-based ETags using modification time and size
- Range request support with If-Range validation

## Running the Example

```bash
cd examples/etag
go run main.go
```

The server starts on http://localhost:8080

## Testing ETags

### 1. Basic ETag Generation

```bash
curl -i http://localhost:8080/api/static-data
```

Response:
```
HTTP/1.1 200 OK
Content-Type: application/json
Etag: W/"7b2c9f4a8e1d"
Content-Length: 38

{"message":"This response has an ETag header"}
```

### 2. Conditional Request (304 Not Modified)

Make the same request with the ETag from the previous response:

```bash
# Copy the ETag from the previous response (e.g., W/"abc123")
curl -i http://localhost:8080/api/static-data -H 'If-None-Match: "abc123"'
```

Response:
```
HTTP/1.1 304 Not Modified
Etag: W/"abc123"
```

### 3. Static File Serving with ETags

```bash
curl -i http://localhost:8080/hello.txt
```

Response:
```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Etag: W/"1709999999-54"
Last-Modified: ...

Hello, World!
This is a sample file for ETag testing.
```

### 4. File Range Requests with If-Range

```bash
# First, get the file's ETag
curl -i http://localhost:8080/large.txt | grep -i etag

# Request a specific range with If-Range validation
# (replace "abc123" with the actual ETag from above)
curl -i http://localhost:8080/large.txt \
  -H 'Range: bytes=0-99' \
  -H 'If-Range: "abc123"'
```

Response (when ETag matches):
```
HTTP/1.1 206 Partial Content
Content-Type: text/plain; charset=utf-8
Content-Range: bytes 0-99/1000
Content-Length: 100

abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv
```

Response (when ETag doesn't match - full content):
```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Length: 1000

[full file content]
```

### 5. File-Based ETag Endpoint

```bash
curl -i http://localhost:8080/file/hello.txt
```

This endpoint uses `middleware.GenerateFileETag()` which creates an ETag
from the file's modification time and size, avoiding the need to hash
the file content.

## ETag Algorithms

The middleware supports two hash algorithms:

- **FNV-1a** (default): Fast, suitable for most use cases
- **MD5**: More collision-resistant, slower

Configure with:

```go
app.Use(middleware.ETag(config.WithETagAlgorithm(config.MD5)))
```

## Weak vs Strong ETags

By default, weak ETags are generated (`W/"..."`):
- Suitable for dynamically generated content
- Allow byte-for-byte differences in semantically equivalent content

For strong ETags (byte-for-byte equality):

```go
app.Use(middleware.ETag(config.WithETagWeak(false)))
```

## Content-Encoding Aware ETags

When used with compression middleware, ETags automatically include
the content encoding in the hash calculation, preventing cache poisoning
where the same ETag could be returned for both compressed and uncompressed content.

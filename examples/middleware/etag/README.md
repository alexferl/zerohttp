# ETag Middleware Example

This example demonstrates zerohttp's ETag middleware for HTTP caching and conditional requests.

## Features

- Automatic ETag generation for responses
- Conditional request handling (304 Not Modified)
- File-based ETags for static files
- Support for weak and strong ETags

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint               | Description                                |
|------------------------|--------------------------------------------|
| `GET /api/data`        | Dynamic JSON with ETag                     |
| `GET /api/static-data` | Static JSON content (good for 304 testing) |
| `GET /static/*`        | Static files with file-based ETags         |
| `GET /file/{name}`     | Custom file handler with ETag              |

## Test Commands

### Basic ETag generation
```bash
curl -i http://localhost:8080/api/static-data
```

### Conditional request (304 Not Modified)
```bash
# First get the ETag from the previous response, then:
curl -i http://localhost:8080/api/static-data -H 'If-None-Match: "YOUR_ETAG"'
```

### Static file with ETag
```bash
curl -i http://localhost:8080/hello.txt
```

### File with conditional request
```bash
curl -i http://localhost:8080/hello.txt -H 'If-None-Match: W/"YOUR_ETAG"'
```

### Range request with If-Range
```bash
curl -i http://localhost:8080/large.txt -H 'Range: bytes=0-99' -H 'If-Range: W/"YOUR_ETAG"'
```

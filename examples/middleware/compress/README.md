# Compression Middleware Example

This example demonstrates zerohttp's built-in compression middleware with gzip and deflate support.

## Features

- Automatic response compression based on Accept-Encoding header
- Support for gzip and deflate algorithms
- JSON and HTML response compression

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint        | Description                             |
|-----------------|-----------------------------------------|
| `GET /`         | HTML page (compressed if supported)     |
| `GET /api/data` | JSON response (compressed if supported) |

## Test Commands

### Request with gzip compression
```bash
curl -i -H "Accept-Encoding: gzip" http://localhost:8080/
```

### Request with deflate compression
```bash
curl -i -H "Accept-Encoding: deflate" http://localhost:8080/
```

### Request JSON with compression
```bash
curl -i -H "Accept-Encoding: gzip" http://localhost:8080/api/data
```

### Request without compression
```bash
curl -i http://localhost:8080/
```

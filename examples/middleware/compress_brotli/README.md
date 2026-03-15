# Brotli Compression Example

This example demonstrates how to add Brotli compression support to zerohttp using the `github.com/andybalholm/brotli` package.

Brotli typically provides 20-26% better compression than gzip.

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Test Commands

### Request with Brotli compression
```bash
curl -i -H "Accept-Encoding: br" http://localhost:8080/
```

### Request with gzip (fallback)
```bash
curl -i -H "Accept-Encoding: gzip" http://localhost:8080/
```

### Request without compression
```bash
curl -i http://localhost:8080/
```

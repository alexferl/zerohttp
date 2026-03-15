# Zstd Compression Example

This example demonstrates how to add Zstandard (zstd) compression support to zerohttp using the `github.com/klauspost/compress` package.

Zstd provides excellent compression ratios with very fast decompression speeds.

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Test Commands

### Request with zstd compression
```bash
curl -i -H "Accept-Encoding: zstd" http://localhost:8080/
```

### Request with gzip (fallback)
```bash
curl -i -H "Accept-Encoding: gzip" http://localhost:8080/
```

### Request without compression
```bash
curl -i http://localhost:8080/
```

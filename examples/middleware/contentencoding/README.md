# Content Encoding Example

This example demonstrates zerohttp's content encoding middleware for validating request content encodings.

## Features

- Rejects requests with unsupported content encodings (returns 415 Unsupported Media Type)
- Allows only specified encodings (e.g., gzip, deflate)
- Skips validation for empty request bodies

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Description                                   |
|------------------|-----------------------------------------------|
| `POST /api/data` | Accepts data with gzip or deflate encoding    |

## Test Commands

### Request with gzip encoding (succeeds)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Encoding: gzip" \
  -H "Content-Type: application/json" \
  --data-binary <(echo '{"message":"hello"}' | gzip)
```

### Request with unsupported encoding (fails with 415)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Encoding: br" \
  -d '{"message":"hello"}'
```

### Request without encoding (succeeds - empty body skipped)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/json" \
  -d '{"message":"hello"}'
```

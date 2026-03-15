# Content Charset Example

This example demonstrates zerohttp's content charset middleware for validating request character encodings.

## Features

- Rejects requests with unsupported charsets (returns 415 Unsupported Media Type)
- Allows only specified charsets (e.g., UTF-8)

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Description                          |
|------------------|--------------------------------------|
| `POST /api/data` | Accepts JSON data with UTF-8 charset |

## Test Commands

### Request with UTF-8 charset (succeeds)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/json; charset=utf-8" \
  -d '{"message":"hello"}'
```

### Request with unsupported charset (fails with 415)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/json; charset=iso-8859-1" \
  -d '{"message":"hello"}'
```

### Request without charset (fails - no charset specified)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/json" \
  -d '{"message":"hello"}'
```

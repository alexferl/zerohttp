# Content Type Example

This example demonstrates zerohttp's content type middleware for validating request Content-Types.

## Features

- Rejects requests with unsupported content types (returns 415 Unsupported Media Type)
- Allows only specified content types (e.g., application/json)
- Skips validation for empty request bodies

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Description                        |
|------------------|------------------------------------|
| `POST /api/data` | Accepts only JSON content type     |

## Test Commands

### Request with JSON content type (succeeds)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/json" \
  -d '{"message":"hello"}'
```

### Request with form data (fails with 415)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "message=hello"
```

### Request without content type (succeeds - empty body skipped)
```bash
curl -X POST http://localhost:8080/api/data
```

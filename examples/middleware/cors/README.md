# CORS Example

This example demonstrates zerohttp's CORS (Cross-Origin Resource Sharing) middleware.

## Features

- Configure allowed origins, methods, and headers
- Support for credentials and exposed headers
- Preflight request handling

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Description                |
|------------------|----------------------------|
| `GET /api/data`  | CORS-enabled GET endpoint  |
| `POST /api/data` | CORS-enabled POST endpoint |

## Test Commands

### Simple CORS request (succeeds from allowed origin)
```bash
curl -i http://localhost:8080/api/data \
  -H "Origin: http://localhost:3000"
```

### CORS preflight request
```bash
curl -i -X OPTIONS http://localhost:8080/api/data \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type"
```

### Request from disallowed origin (no CORS headers returned)
```bash
curl -i http://localhost:8080/api/data \
  -H "Origin: https://evil.com"
```

Note: CORS is a browser security mechanism. The server still processes the request and returns data (as you see with curl), but browsers will block the response if CORS headers are missing or don't match the origin.

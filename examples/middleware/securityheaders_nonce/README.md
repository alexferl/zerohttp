# CSP Nonce Example

This example demonstrates Content Security Policy (CSP) nonce generation for allowing inline scripts and styles.

## Features

- Unique nonce generated per request
- Inline scripts and styles validated via nonce
- CSP header with nonce placeholder replacement

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Description                                         |
|------------------|-----------------------------------------------------|
| `GET /`          | Demo page with inline scripts using CSP nonce       |
| `GET /api/nonce` | Returns the current request's nonce (for debugging) |

## Test Commands

### View the demo page
```bash
curl -i http://localhost:8080/
```

### Get the current nonce
```bash
curl http://localhost:8080/api/nonce
```

### Check CSP header
```bash
curl -I http://localhost:8080/ | grep -i content-security-policy
```

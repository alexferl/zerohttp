# Request ID Example

This example demonstrates zerohttp's request ID middleware for tracing requests through your application and logs.

> **Note:** The RequestID middleware is included **by default** in all zerohttp applications. This example shows how to access the request ID and customize the configuration if needed.

## Features

- Automatic request ID generation using crypto-secure random (128 bits entropy)
- Request ID propagation via HTTP headers
- Request ID storage in context for use in handlers
- Custom generators and header names

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint         | Description                                    |
|------------------|------------------------------------------------|
| `GET /`          | Returns the request ID from context            |
| `GET /headers`   | Shows request ID in response header            |
| `GET /custom`    | Uses custom generator (UUID-like format)       |

## Test Commands

### Basic request (auto-generated ID)
```bash
curl -i http://localhost:8080/
```

Response:
```json
{
  "message": "Hello!",
  "request_id": "a1b2c3d4e5f6..."
}
```

Response headers include:
```
X-Request-Id: a1b2c3d4e5f6...
```

### Provide your own request ID
```bash
curl -i http://localhost:8080/ -H "X-Request-Id: my-custom-id-123"
```

The provided ID is returned in the response and propagated to the context.

### Custom generator endpoint
```bash
curl -i http://localhost:8080/custom
```

Uses a custom generator that prefixes IDs with "custom-".

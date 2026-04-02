# Custom Middleware Example

This example demonstrates how to write custom middleware for zerohttp.

## Features

- Basic middleware pattern
- Factory function for configurable middleware
- Global middleware application

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint           | Description                     |
|--------------------|---------------------------------|
| `GET /`            | Returns a simple message        |
| `GET /hello/{name}` | Returns a personalized greeting |

## Test Commands

### Basic request
```bash
curl -i http://localhost:8080/
```

Shows response headers:
```
X-Custom-Version: 1.0
```

### Personalized greeting
```bash
curl http://localhost:8080/hello/world
```

Response:
```json
{
  "message": "Hello world"
}
```

## How It Works

The example shows two middleware patterns:

1. **Simple Middleware** (`requestTimer`): Logs request timing information
2. **Factory Function** (`addHeader`): Creates middleware with custom configuration

Both follow the standard Go middleware signature:
```go
func(http.Handler) http.Handler
```

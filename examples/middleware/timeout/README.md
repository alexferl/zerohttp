# Timeout Example

This example demonstrates zerohttp's timeout middleware for limiting request processing time.

## Features

- Request timeout with configurable duration
- Custom timeout message
- Context cancellation for cleanup

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint    | Description                           |
|-------------|---------------------------------------|
| `GET /fast` | Returns immediately                   |
| `GET /slow` | Random 1-4 second delay (may timeout) |

## Test Commands

### Fast request (no timeout)
```bash
curl http://localhost:8080/fast
```

### Slow request (may timeout)
```bash
curl http://localhost:8080/slow
```

If the response takes longer than 2 seconds:
```
Request timeout
```

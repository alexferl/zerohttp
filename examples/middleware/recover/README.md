# Recover Example

This example demonstrates zerohttp's panic recovery middleware. Recover is included in the default middlewares, so your server automatically recovers from panics without crashing.

## Features

- Automatic panic recovery (enabled by default)
- Stack trace logging
- Request ID correlation
- Metrics tracking (`recover_panics_total`)
- Returns HTTP 500 for panics

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint              | Description                   |
|-----------------------|-------------------------------|
| `GET /healthy`        | Normal endpoint (no panic)    |
| `GET /panic/nil`      | Nil pointer dereference panic |
| `GET /panic/explicit` | Explicit panic() call         |
| `GET /panic/error`    | Panic with error type         |

## Test Commands

### Healthy endpoint (works normally)
```bash
curl -i http://localhost:8080/healthy
```

### Nil pointer panic (returns 500, server keeps running)
```bash
curl -i http://localhost:8080/panic/nil
```

Response:
```json
{
  "status": 500,
  "title": "Internal Server Error",
  "detail": "Internal server error"
}
```

### Explicit panic (returns 500, server keeps running)
```bash
curl -i http://localhost:8080/panic/explicit
```

### Error panic (returns 500, server keeps running)
```bash
curl -i http://localhost:8080/panic/error
```

### Verify server is still running after panics
```bash
curl http://localhost:8080/healthy
```

## Default Behavior

The Recover middleware is automatically included via `DefaultMiddlewares()`:

1. **Catches panics** - Any panic in a handler is caught
2. **Logs the error** - Includes panic value and stack trace
3. **Returns 500** - Sends a proper HTTP 500 response
4. **Tracks metrics** - Increments `recover_panics_total` counter
5. **Preserves request ID** - Correlates panic with request

## Custom Configuration

To customize recover behavior, add it manually:

```go
app.Use(recover.New(logger, recover.Config{
    EnableStackTrace: true,   // Include stack traces in logs
    StackSize:        4096,   // Stack trace buffer size (bytes)
    RequestIDHeader:  "X-Request-Id",
}))
```

Or disable default middlewares and add your own:

```go
app := zh.New(zh.Config{
    DisableDefaultMiddlewares: true,
})
app.Use(recover.New(app.Logger(), recover.DefaultConfig))
```

## Security Note

The Recover middleware ensures your server stays running even when handlers panic. However, you should still:

- Monitor `recover_panics_total` metrics for unexpected panics
- Fix underlying bugs causing panics
- Use structured error handling instead of panic()

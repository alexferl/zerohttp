# Request Body Size Example

This example demonstrates zerohttp's request body size limiting middleware to prevent large uploads and protect against denial-of-service attacks.

## Features

- Limit request body size globally via server config
- Automatic 413 response for oversized bodies
- Metrics tracking for rejected requests

## Important Note

The body size limit is enforced by `http.MaxBytesReader`, which only limits **how much can be read** from the request body. Your handler must actually read `r.Body` (e.g., with `io.ReadAll` or `json.NewDecoder`) for the limit to be enforced.

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint            | Description                     |
|---------------------|---------------------------------|
| `POST /api/data`    | Regular endpoint (100KB limit)  |
| `POST /api/webhook` | Webhook endpoint (100KB limit)  |

## Test Commands

### Small payload (succeeds)
```bash
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: application/json" \
  -d '{"message": "small data"}'
```

### Large payload (rejected with 413)
```bash
# Create a 200KB payload
dd if=/dev/zero bs=1024 count=200 2>/dev/null | base64 > /tmp/large.txt
curl -X POST http://localhost:8080/api/data \
  -H "Content-Type: text/plain" \
  --data-binary @/tmp/large.txt
```

Response:
```json
{
  "title": "Payload Too Large",
  "status": 413,
  "detail": "Request body exceeds maximum allowed size"
}
```

## Configuration

Configure via server config (affects the default middleware):

```go
app := zh.New(config.Config{
    RequestBodySize: config.RequestBodySizeConfig{
        MaxBytes: 100 * 1024, // 100KB
    },
})
```

### Exempting Paths

```go
app := zh.New(config.Config{
    RequestBodySize: config.RequestBodySizeConfig{
        MaxBytes:    1 * 1024 * 1024, // 1MB
        ExemptPaths: []string{"/api/webhook", "/health"},
    },
})
```

## Important Behavior Notes

### Handler Continues After 413
When the body size limit is exceeded, the middleware sends a 413 response but **the handler continues executing**. The handler will receive the `*http.MaxBytesError` from `r.Body.Read()`. You should check for this error and return early:

```go
app.POST("/api/data", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        // Middleware already sent 413, just return the error
        return err
    }
    // Process body...
}))
```

### Automatic 413 Response
The middleware automatically writes the 413 status when the limit is exceeded. If your handler tries to write a different response after the error, it will be suppressed (no superfluous WriteHeader warnings).

## Default Behavior

The RequestBodySize middleware is included by default with a 1MB limit.

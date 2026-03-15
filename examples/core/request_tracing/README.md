# Request Tracing Example

Demonstrates request ID propagation for distributed tracing and logging.

## Running

```bash
go run .
```

## Endpoints

- `GET /trace` - Returns request ID and duration
- `GET /chain` - Simulates distributed tracing across services

## Request ID

The `RequestID` middleware is enabled by default and adds an `X-Request-ID` header to all responses. Access the request ID in handlers using:

```go
requestID := middleware.GetRequestID(r.Context())
```

Use the request ID for:
- Correlating logs across a request lifecycle
- Propagating trace context to downstream services
- Debugging and request tracking

## Example Output

```bash
$ curl -i http://localhost:8080/trace
HTTP/1.1 200 OK
X-Request-Id: abc123-def456
Content-Type: application/json

{
  "request_id": "abc123-def456",
  "duration_ms": 15,
  "message": "Request traced successfully"
}
```

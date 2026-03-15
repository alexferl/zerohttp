# Circuit Breaker Example

This example demonstrates zerohttp's circuit breaker middleware for protecting against cascading failures.

## Features

- Automatic circuit breaking after consecutive failures
- Recovery timeout with half-open state
- Per-endpoint circuit isolation

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint       | Description                                                  |
|----------------|--------------------------------------------------------------|
| `GET /flaky`   | Fails first 5 requests, then works (to demo circuit breaker) |
| `GET /healthy` | Always returns 200 OK                                        |

## Test Commands

### Test the flaky endpoint (will fail first 5 times)
```bash
for i in {1..7}; do curl -s http://localhost:8080/flaky; echo; done
```

### After circuit opens, requests are rejected immediately
```bash
curl -i http://localhost:8080/flaky
```

### Healthy endpoint is unaffected
```bash
curl -i http://localhost:8080/healthy
```

## Circuit Breaker States

1. **Closed** - Normal operation, requests pass through
2. **Open** - After 3 failures, rejects requests with 503
3. **Half-Open** - After 5s timeout, allows 1 test request

# Graceful Shutdown Example

This example demonstrates graceful server shutdown with signal handling.

## Features

- Signal handling for graceful shutdown
- Configurable shutdown timeout (5 seconds)
- Proper connection draining

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint | Description                              |
|--------|----------|------------------------------------------|
| `GET`  | `/`      | Hello world                              |
| `GET`  | `/fast`  | Completes in 2s (before timeout)         |
| `GET`  | `/slow`  | Completes in 10s (after timeout)         |

## Test Commands

### Test the endpoints
```bash
curl http://localhost:8080/
curl http://localhost:8080/fast
curl http://localhost:8080/slow
```

### Demonstrate graceful shutdown

1. Start the server:
```bash
go run .
```

2. In another terminal, start a slow request:
```bash
time curl http://localhost:8080/slow
```

3. While the slow request is running, press `Ctrl+C` in the first terminal

4. The server will wait 5 seconds for the request to complete before forcing shutdown

### Behavior

**Fast requests** (`/fast`): Complete normally during shutdown (2s < 5s timeout)

**Slow requests** (`/slow`): These take 10 seconds, which exceeds the 5-second shutdown timeout. When you trigger shutdown while a slow request is running:
- The server waits 5 seconds for the request to complete
- After the timeout, the connection is force-closed
- The client receives a connection error
- The handler continues running in the background (server doesn't kill goroutines)

This demonstrates that handlers running longer than the shutdown timeout will be force-terminated.

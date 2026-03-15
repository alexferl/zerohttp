# Lifecycle Hooks Example

This example demonstrates startup and shutdown lifecycle hooks.

## Features

- **Pre-startup hooks** - Run before servers start
- **Startup hooks** - Run concurrently with servers starting up
- **Post-startup hooks** - Run after servers have started
- **Pre-shutdown hooks** - Run before shutdown begins
- **Shutdown hooks** - Run concurrently with server shutdown
- **Post-shutdown hooks** - Run after servers have shut down

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint | Description   |
|--------|----------|---------------|
| `GET`  | `/`      | Hello world   |

## Hook Execution Order

### Startup

1. **Pre-startup hooks** - Validate configuration, check prerequisites
2. **Servers start** - HTTP/HTTPS servers begin accepting connections
3. **Startup hooks** - Initialize resources (DB, cache) concurrently with server startup
4. **Post-startup hooks** - Announce readiness, register with service discovery

### Shutdown (when you press `Ctrl+C`)

1. **Pre-shutdown hooks** - Mark service unhealthy, stop accepting new work
2. **Servers shutdown** - Stop accepting new connections, wait for requests to complete
3. **Shutdown hooks** - Close resources (DB, cache, connections) concurrently
4. **Post-shutdown hooks** - Final cleanup after servers are stopped

## Test Commands

```bash
curl http://localhost:8080/
```

Press `Ctrl+C` to see the hooks execute during shutdown.

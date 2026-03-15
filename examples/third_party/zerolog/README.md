# Zerolog Integration Example

This example demonstrates how to integrate [Zerolog](https://github.com/rs/zerolog) (a high-performance structured logging library) with zerohttp.

## Features

- Structured JSON logging with Zerolog
- Console output with pretty formatting
- Adapter pattern for zerohttp's Logger interface
- Field type handling (string, int, error, bool, etc.)

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Returns a JSON response and logs an info message |

## Log Output

When you visit `http://localhost:8080`, you'll see structured log output:

```
2026-03-14T10:30:00Z |INFO| main.go:27 > I'm a log! method=GET path=/ status=200
```

## Project Structure

- `main.go` - Server setup with Zerolog logger injection
- `adapter.go` - Zerolog adapter implementing zerohttp's Logger interface

## How It Works

The `ZerologAdapter` wraps a `zerolog.Logger` and implements zerohttp's `log.Logger` interface:

```go
type ZerologAdapter struct {
    logger zerolog.Logger
}
```

It implements methods like `Debug()`, `Info()`, `Warn()`, `Error()`, etc., mapping them to zerolog's equivalent methods.

The adapter is passed to zerohttp via config:

```go
zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
    Level(zerolog.InfoLevel).
    With().
    Timestamp().
    Caller().
    Logger()
logger := NewZerologAdapter(zl)

app := zh.New(config.Config{
    Logger: logger,
})
```

Now all zerohttp logging (request logging, errors, etc.) uses Zerolog for output.

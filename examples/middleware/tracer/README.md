# Tracing Example

This example demonstrates request tracing with a custom tracer.

## Features

- Request tracing with spans
- Custom tracer implementation
- Error tracking

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint     | Description        |
|--------------|--------------------|
| `GET /`      | Successful request |
| `GET /error` | Request with error |

## Test Commands

```bash
curl http://localhost:8080/
curl http://localhost:8080/error
```

Watch the console for trace output.

# Problem Detail Example

This example demonstrates RFC 7807 Problem Details for HTTP APIs - a standardized way to express errors.

## Features

- RFC 7807 compliant error responses
- ProblemDetail with type, title, status, detail, instance
- Validation error responses
- Custom error types

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Method | Endpoint           | Description                        |
|--------|--------------------|------------------------------------|
| `GET`  | `/`                | Hello world                        |
| `GET`  | `/error`           | Standard ProblemDetail error       |
| `POST` | `/validate-simple` | Validation errors (default format) |
| `POST` | `/validate-custom` | Validation errors (custom format)  |

## Test Commands

### Basic endpoint
```bash
curl http://localhost:8080/
```

### ProblemDetail error
```bash
curl http://localhost:8080/error
```

### Validation errors (simple)
```bash
curl -X POST http://localhost:8080/validate-simple
```

### Validation errors (custom)
```bash
curl -X POST http://localhost:8080/validate-custom
```

## ProblemDetail Format

Standard RFC 7807 response:

```json
{
  "type": "https://example.com/probs/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "The requested resource was not found",
  "instance": "/error"
}
```

Validation errors:

```json
{
  "type": "about:blank",
  "title": "Validation Failed",
  "status": 422,
  "detail": "Validation failed",
  "errors": [
    {"detail": "must be positive", "pointer": "#/age"},
    {"detail": "invalid email", "field": "email"}
  ]
}
```

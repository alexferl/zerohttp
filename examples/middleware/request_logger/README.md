# Request Logger Example

This example demonstrates zerohttp's request logging middleware with body logging and automatic masking of sensitive fields.

> **Note:** The RequestLogger middleware is included **by default** in all zerohttp applications. This example shows how to enable body logging and configure fields.

## Features

- Request/response body logging
- Automatic masking of sensitive fields (password, token, etc.)
- Configurable log fields
- Body size limits

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint          | Description                        |
|-------------------|------------------------------------|
| `POST /api/login` | Login endpoint with token response |
| `GET /health`     | Health check endpoint              |

## Test Commands

### Login (password field will be masked)
```bash
curl -X POST http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"john","password":"super_secret_123"}'
```

### Health check
```bash
curl http://localhost:8080/health
```

## Configuration

```go
app := zh.New(config.Config{
    RequestLogger: config.RequestLoggerConfig{
        LogRequestBody:  true,
        LogResponseBody: true,
        MaxBodySize:     1024,
        Fields: []config.LogField{
            config.FieldMethod,
            config.FieldPath,
            config.FieldStatus,
            config.FieldDurationHuman,
            config.FieldRequestBody,
            config.FieldResponseBody,
        },
    },
})
```

## Sensitive Field Masking

The logger automatically masks sensitive fields in request/response bodies:

- `password`, `passwd`, `pwd`
- `token`, `access_token`, `refresh_token`
- `secret`, `api_key`, `apikey`
- `authorization`, `auth`
- `credit_card`, `cvv`, `ssn`

Masked values appear as `[REDACTED]` in logs.

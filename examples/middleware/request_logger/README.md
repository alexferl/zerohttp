# Request Logger Example

This example demonstrates zerohttp's request logging middleware with body logging, automatic masking of sensitive fields, and custom fields.

> **Note:** The RequestLogger middleware is included **by default** in all zerohttp applications. This example shows how to enable body logging, configure fields, and add custom fields.

## Features

- Request/response body logging
- Automatic masking of sensitive fields (password, token, etc.)
- Configurable log fields
- Body size limits
- Custom fields

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint           | Description                        |
|--------------------|------------------------------------|
| `POST /api/login`  | Login endpoint with token response |
| `GET /admin/users` | Admin endpoint (adds access_level) |
| `GET /health`      | Health check endpoint              |

## Test Commands

### Login with API key (logs tenant_id, masks password)
```bash
curl -X POST http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: sk-1234567890abcdef' \
  -d '{"username":"john","password":"super_secret_123"}'
```

### Access admin endpoint (logs tenant_id and access_level)
```bash
curl http://localhost:8080/admin/users \
  -H 'X-API-Key: sk-abcdef1234567890'
```

### Health check
```bash
curl http://localhost:8080/health
```

## Features

### Sensitive Field Masking

The logger automatically masks sensitive fields in request/response bodies:

- `password`, `passwd`, `pwd`
- `token`, `access_token`, `refresh_token`
- `secret`, `api_key`, `apikey`
- `authorization`, `auth`
- `credit_card`, `cvv`, `ssn`

Masked values appear as `[REDACTED]` in logs.

### Custom Fields Use Cases

- **API Key Logging**: Track which API keys are being used
- **User Identification**: Log user IDs from JWT claims or sessions
- **Tenant/Org Tracking**: Multi-tenant application support
- **Request Classification**: Internal vs external requests
- **Custom Correlation IDs**: Link related requests

Return `nil` or an empty slice when no custom fields are needed for a request.

# Request Logger Example

This example demonstrates zerohttp's request logging middleware with body logging, automatic masking of sensitive fields, and custom fields.

> **Note:** The RequestLogger middleware is included **by default** in all zerohttp applications. This example shows how to enable body logging, configure fields, and add custom fields.

## Features

- Request/response body logging
- Automatic masking of sensitive fields (password, token, etc.)
- Configurable log fields
- Body size limits
- **Custom fields from headers and context**

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

## Configuration

### Basic Configuration

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

### Custom Fields

The `CustomFields` callback allows you to add arbitrary fields to request logs:

```go
// Simulate a database lookup of tenant ID from API key
var apiKeyToTenant = map[string]string{
    "sk-1234567890abcdef": "tenant-acme",
    "sk-abcdef1234567890": "tenant-cyberdyne",
}

app := zh.New(config.Config{
    RequestLogger: config.RequestLoggerConfig{
        // ... other config ...
        CustomFields: func(r *http.Request) []log.Field {
            var fields []log.Field

            // Look up tenant ID from API key (don't log the key itself!)
            if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
                if tenantID, ok := apiKeyToTenant[apiKey]; ok {
                    fields = append(fields, log.F("tenant_id", tenantID))
                }
            }

            // Conditional fields based on path
            if strings.HasPrefix(r.URL.Path, "/admin/") {
                fields = append(fields, log.F("access_level", "admin"))
            }

            return fields
        },
    },
})
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

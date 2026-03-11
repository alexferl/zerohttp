# Request Logger with Body Logging Example

This example demonstrates the request/response body logging feature in zerohttp with automatic masking of sensitive fields.

## Features Demonstrated

- **Request body logging**: Captures and logs incoming request bodies
- **Response body logging**: Captures and logs outgoing response bodies
- **Sensitive field masking**: Automatically masks fields like `password`, `token`, `secret`, etc.
- **Max body size limit**: Configurable limit to prevent excessive logging
- **Nested object support**: Masking works in nested JSON objects

## Running the Example

```bash
go run main.go
```

## Testing the Endpoints

### 1. Login (password field masked)

```bash
curl -X POST http://localhost:8080/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"john","password":"super_secret_123"}'
```

**Expected log output:**
```
request_body={"username":"john","password":"[REDACTED]"}
```

### 2. Get Token (tokens masked in response)

```bash
curl -X POST http://localhost:8080/api/token \
  -H 'Content-Type: application/json' \
  -d '{"grant_type":"password"}'
```

**Expected log output:**
```
response_body={"access_token":"[REDACTED]","refresh_token":"[REDACTED]","token_type":"Bearer"}
```

### 3. Create User (nested objects)

```bash
curl -X POST http://localhost:8080/api/users \
  -H 'Content-Type: application/json' \
  -d '{"username":"jane","email":"jane@example.com"}'
```

### 4. Health Check

```bash
curl http://localhost:8080/health
```

## Configuration

The middleware is configured with:

```go
middleware.RequestLogger(logger, config.RequestLoggerConfig{
    LogRequestBody:  true,                    // Enable request body logging
    LogResponseBody: true,                    // Enable response body logging
    MaxBodySize:     1024,                    // Max bytes to log (1KB)
    Fields: []config.LogField{
        config.FieldMethod,
        config.FieldPath,
        config.FieldStatus,
        config.FieldDurationHuman,
        config.FieldRequestBody,
        config.FieldResponseBody,
    },
})
```

## Default Sensitive Fields

The following fields are automatically masked by default:

- `password`, `passwd`, `pwd`
- `secret`, `token`, `api_key`, `apikey`
- `access_token`, `refresh_token`, `id_token`
- `authorization`, `auth`
- `credential`, `credentials`
- `private_key`, `privatekey`, `ssh_key`, `sshkey`
- `credit_card`, `creditcard`, `cc_number`, `cvv`
- `ssn`, `dob`

## Custom Sensitive Fields

You can customize which fields to mask:

```go
middleware.RequestLogger(logger, config.RequestLoggerConfig{
    LogRequestBody:  true,
    LogResponseBody: true,
    SensitiveFields: []string{"ssn", "credit_card", "pin"},
})
```

To disable masking entirely, set `SensitiveFields` to an empty slice:

```go
SensitiveFields: []string{}, // Empty but not nil
```

## Performance Considerations

- Body logging is **opt-in** and disabled by default
- Request bodies are read and restored transparently
- Response bodies are captured without affecting the response
- Max body size limits prevent excessive memory usage
- Non-JSON bodies are logged as-is without masking

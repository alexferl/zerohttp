# HMAC Authentication Example

This example demonstrates HMAC request signing authentication with zerohttp.

## Features

- HMAC-SHA256 request signing
- Presigned URL support for downloads
- Multiple service credentials

## Running the Example

Start the server:
```bash
go run .
```

In another terminal, run the client:
```bash
go run ./client
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint            | Description                                  |
|---------------------|----------------------------------------------|
| `GET /health`       | Public health check (no auth)                |
| `GET /api/data`     | Protected API endpoint (HMAC auth required)  |
| `GET /api/download` | Protected download (supports presigned URLs) |

## Credentials

- Access Key: `service-a` / Secret: `super-secret-key-at-least-32-bytes-long!!`
- Access Key: `service-b` / Secret: `another-secret-key-for-service-b-abc123`

## Test Commands

### Public endpoint (no auth)
```bash
curl http://localhost:8080/health
```

### Without authentication (fails with 401)
```bash
curl -i http://localhost:8080/api/data
```

### With HMAC authentication
The middleware requires these headers:
- `Authorization: HMAC <access-key>:<signature>`
- `X-Date`: RFC3339 timestamp

Generate a signature (requires the Go client code):
```go
signer := middleware.NewHMACSigner("service-a", "super-secret-key-at-least-32-bytes-long!!")
signer.SignRequest(req)
```

Or use the presigned URL feature for simple GET requests:
```go
signer := middleware.NewHMACSigner("service-a", "super-secret-key-at-least-32-bytes-long!!")
req, _ := http.NewRequest(http.MethodGet, "http://localhost:8080/api/download", nil)
url, _ := signer.PresignURL(req, 5*time.Minute)
// Use url.String() for the request
```

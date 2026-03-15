# Host Validation Example

This example demonstrates zerohttp's host validation middleware for protecting against DNS rebinding and Host header attacks.

## Features

- Exact host matching
- Subdomain allowlisting
- Strict port validation
- Exempt paths
- Custom error responses

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint               | Allowed Hosts        | Description            |
|------------------------|----------------------|------------------------|
| `GET /api/basic`       | `api.example.com`    | Exact match only       |
| `GET /api/subdomains`  | `*.example.com`      | Subdomains allowed     |
| `GET /api/multi`       | Multiple hosts       | Multiple allowed hosts |
| `GET /api/strict-port` | `localhost:8080`     | Requires port in Host  |
| `GET /health`          | Any (exempt)         | Bypasses validation    |
| `GET /api/custom`      | `secure.example.com` | Custom error response  |

## Test Commands

### Basic validation (only api.example.com)
```bash
curl -H "Host: api.example.com" http://localhost:8080/api/basic
curl -H "Host: evil.com" http://localhost:8080/api/basic  # rejected
```

### Subdomains allowed
```bash
curl -H "Host: api.example.com" http://localhost:8080/api/subdomains
curl -H "Host: v1.api.example.com" http://localhost:8080/api/subdomains
```

### Strict port validation
```bash
curl -H "Host: localhost:8080" http://localhost:8080/api/strict-port
curl -H "Host: localhost" http://localhost:8080/api/strict-port  # rejected
```

### Exempt path (no validation)
```bash
curl -H "Host: anything.com" http://localhost:8080/health
```

# Security Headers Example

This example demonstrates zerohttp's security headers middleware for protecting against common web vulnerabilities.

> **Note:** The SecurityHeaders middleware is included **by default** in all zerohttp applications with secure defaults.

## Features

- Content Security Policy (CSP)
- X-Frame-Options (clickjacking protection)
- X-Content-Type-Options (MIME sniffing protection)
- Referrer-Policy
- Permissions-Policy

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint        | Description                        |
|-----------------|------------------------------------|
| `GET /`         | Default security headers           |
| `GET /api/docs` | Relaxed CSP for API documentation  |

## Test Commands

### Check default security headers
```bash
curl -I http://localhost:8080/
```

Response headers include:
```
Content-Security-Policy: default-src 'none'; script-src 'self'; ...
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

### Relaxed CSP
```bash
curl -I http://localhost:8080/api/docs
```

Shows relaxed CSP allowing inline styles/scripts (useful for Swagger, etc.).

## CSP Nonce Example

For CSP nonce support, see the dedicated example:

```bash
go run examples/middleware/security_headers_nonce/main.go
```

## Default Security Headers

| Header                  | Default Value                                           |
|-------------------------|---------------------------------------------------------|
| Content-Security-Policy | `default-src 'none'; script-src 'self'; ...`            |
| X-Frame-Options         | `DENY`                                                  |
| X-Content-Type-Options  | `nosniff`                                               |
| Referrer-Policy         | `no-referrer`                                           |
| Permissions-Policy      | Restrictive defaults (no camera, mic, geolocation, ...) |

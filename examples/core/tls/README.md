# TLS/HTTPS Example

This example demonstrates configuring HTTPS/TLS with automatic HTTP to HTTPS redirect.

## Features

- HTTPS server on port 8443
- HTTP redirect middleware
- TLS certificate configuration

## Prerequisites

Generate a self-signed certificate (for development only):

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=localhost"
```

## Running the Example

```bash
go run .
```

The server starts:
- HTTP on `http://localhost:8080` (redirects to HTTPS)
- HTTPS on `https://localhost:8443`

## Test Commands

### HTTPS request (requires -k for self-signed cert)
```bash
curl -k https://localhost:8443
```

### HTTP request (should redirect to HTTPS)
```bash
curl -k -L http://localhost:8080
```

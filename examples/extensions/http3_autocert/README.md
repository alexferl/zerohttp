# HTTP/3 with AutoTLS Example

This example demonstrates HTTP/3 with automatic Let's Encrypt certificate provisioning.

## Features

- HTTP/3 support via quic-go
- Automatic TLS certificates via Let's Encrypt
- HTTP to HTTPS redirect

## Prerequisites

1. A publicly accessible server with a domain name
2. Ports 80 and 443 open
3. Dependencies:
   ```bash
   go mod tidy
   ```

## Running the Example

```bash
go run . -domain example.com
```

## How It Works

- **HTTP on port 80**: ACME challenges and redirects to HTTPS
- **HTTPS on port 443**: HTTP/1, HTTP/2, and HTTP/3 with auto-provisioned certificates

## Test Commands

```bash
curl --http3 https://your-domain.com
```

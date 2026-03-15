# AutoTLS Example

This example demonstrates automatic TLS certificate provisioning using Let's Encrypt via `golang.org/x/crypto/acme/autocert`.

## Features

- Automatic Let's Encrypt certificate issuance
- Certificate caching
- HTTP to HTTPS redirect

## Prerequisites

1. A publicly accessible server with a domain name
2. Ports 80 and 443 open
3. The `golang.org/x/crypto` package:
   ```bash
   go get golang.org/x/crypto/acme/autocert
   ```

## Configuration

Update the `hosts` slice with your domain(s):

```go
var hosts = []string{
    "example.com",
    "www.example.com",
}
```

## Running the Example

```bash
go mod tidy
go run .
```

## How It Works

1. **HTTP on port 80**: Handles ACME challenges and redirects to HTTPS
2. **HTTPS on port 443**: Serves your application with auto-provisioned certificates
3. **Certificate cache**: Stores certs in `/var/cache/certs` to avoid re-issuance

## Test Commands

Once deployed with a real domain:

```bash
curl https://your-domain.com
```

## Security Notes

- Always use a persistent cache directory in production
- Consider restricting file permissions on the cache directory (e.g., `0700`)
- The autocert manager handles certificate renewal automatically

# AutoTLS Example

This example demonstrates automatic TLS certificate provisioning using Let's Encrypt via `golang.org/x/crypto/acme/autocert`.

## Features

- Automatic Let's Encrypt certificate issuance
- Certificate caching
- HTTP to HTTPS redirect

## Prerequisites

- A publicly accessible server with a domain name
- Ports 80 and 443 open
- The `golang.org/x/crypto` package:
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

The server starts:
- HTTP on port 80 (ACME challenges and redirects to HTTPS)
- HTTPS on port 443 (auto-provisioned certificates)

## Test Commands

Once deployed with a real domain:

```bash
curl https://your-domain.com
```

## Security Notes

- Always use a persistent cache directory in production
- Consider restricting file permissions on the cache directory (e.g., `0700`)
- The autocert manager handles certificate renewal automatically

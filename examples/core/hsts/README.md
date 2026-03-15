# HSTS Example

This example shows how to enable HTTP Strict Transport Security (HSTS) in zerohttp.

## What is HSTS?

HSTS tells browsers to always use HTTPS for your site, preventing:
- SSL stripping attacks
- Man-in-the-middle attacks
- Accidental HTTP connections

## Running the Example

```bash
# Generate self-signed certificates (for testing)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=localhost"

# Run the server
go run .
```

## Verify HSTS Header

```bash
# Check that HSTS header is present on HTTPS response
curl -s -D - https://localhost:8443/ -k | grep -i strict-transport-security
```

Expected output:
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

## Configuration Options

```go
StrictTransportSecurity: config.StrictTransportSecurity{
    MaxAge:            31536000, // Time in seconds (1 year = 31536000)
    ExcludeSubdomains: false,    // false = include subdomains (includeSubDomains directive)
    PreloadEnabled:    false,    // true = add preload directive (submit to hstspreload.org)
}
```

## Important Notes

- HSTS only applies to HTTPS responses (header is ignored over HTTP)
- Once browsers cache the HSTS policy, they will refuse HTTP connections
- Test thoroughly before enabling in production
- For preload submission, set `PreloadEnabled: true` and visit https://hstspreload.org/

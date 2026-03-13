# Host Validation Example

This example demonstrates Host header validation with zerohttp.

## What is Host Header Validation?

Host header validation helps prevent:

- **DNS Rebinding Attacks** - Attacker points their domain to your server's IP
- **Virtual Host Confusion** - Requests targeting wrong application
- **Security Policy Bypass** - Host-based access controls

## Running the Example

```bash
go run main.go
```

## Examples

### 1. Basic Host Validation

Only exact matches allowed:

```bash
# Allowed
curl -H 'Host: api.example.com' http://localhost:8080/api/basic

# Rejected
curl -H 'Host: evil.com' http://localhost:8080/api/basic
```

### 2. Subdomain Support

Allow any subdomain of specified hosts:

```bash
# All allowed
curl -H 'Host: api.example.com' http://localhost:8080/api/subdomains
curl -H 'Host: v1.api.example.com' http://localhost:8080/api/subdomains
curl -H 'Host: www.example.com' http://localhost:8080/api/subdomains
```

### 3. Multiple Allowed Hosts

```bash
# Both allowed
curl -H 'Host: api.example.com' http://localhost:8080/api/multi
curl -H 'Host: app.example.com' http://localhost:8080/api/multi
```

### 4. Strict Port Validation

Require Host header to include port when running on non-standard ports:

```bash
# Allowed (port matches)
curl -H 'Host: localhost:8080' http://localhost:8080/api/strict-port

# Rejected (port missing)
curl -H 'Host: localhost' http://localhost:8080/api/strict-port
```

### 5. Exempt Paths

Health checks can bypass validation:

```bash
# Works with any Host header
curl -H 'Host: anything.com' http://localhost:8080/health
```

### 6. Custom Error Response

```bash
# Returns 403 Forbidden with custom message
curl -H 'Host: evil.com' http://localhost:8080/api/custom
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `AllowedHosts` | List of allowed host values (without ports) | `[]` (disabled) |
| `AllowSubdomains` | Allow subdomains of allowed hosts | `false` |
| `StrictPort` | Require port in Host header for non-standard ports | `false` |
| `Port` | Server port (required when `StrictPort: true`) | `0` |
| `StatusCode` | HTTP status for rejected requests | `400` |
| `Message` | Error message for rejected requests | `"Invalid Host header"` |
| `ExemptPaths` | Paths to skip validation | `[]` |

## IPv6 Support

The middleware properly handles IPv6 addresses:

```bash
# With port
curl -H 'Host: [::1]:8080' http://localhost:8080/api/basic

# Without port
curl -H 'Host: ::1' http://localhost:8080/api/basic

# Bracketed without port
curl -H 'Host: [::1]' http://localhost:8080/api/basic
```

# Basic Auth Example

This example demonstrates HTTP Basic Authentication with zerohttp.

## Features

- HTTP Basic Authentication middleware
- Multiple user credentials
- Protected routes

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint        | Description               |
|-----------------|---------------------------|
| `GET /`         | Protected welcome message |
| `GET /api/data` | Protected API data        |

## Test Commands

### Without credentials (will fail with 401)
```bash
curl -i http://localhost:8080/
```

### With valid credentials
```bash
curl -i http://localhost:8080/ -u admin:secret
```

### With different user
```bash
curl -i http://localhost:8080/api/data -u user:password
```

### Wrong password (will fail with 401)
```bash
curl -i http://localhost:8080/ -u admin:wrongpassword
```

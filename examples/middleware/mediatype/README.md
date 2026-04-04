# Media Type Example

This example demonstrates media type negotiation and validation with zerohttp.

## Features

- Accept header validation against allowed media types
- Vendor media type support (+json suffix matching)
- Default media type for clients that accept anything (*/*)
- Versioned API responses based on Accept header

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint        | Description                   |
|-----------------|-------------------------------|
| `GET /api/users` | Returns users (versioned)    |

## Test Commands

### With default version (no Accept header)
```bash
curl -i http://localhost:8080/api/users
```

### With V1 media type
```bash
curl -i http://localhost:8080/api/users \
  -H 'Accept: application/vnd.api.v1+json'
```

### With V2 media type
```bash
curl -i http://localhost:8080/api/users \
  -H 'Accept: application/vnd.api.v2+json'
```

### With unsupported media type (will fail with 406)
```bash
curl -i http://localhost:8080/api/users \
  -H 'Accept: application/xml'
```

### With */* (gets default version)
```bash
curl -i http://localhost:8080/api/users \
  -H 'Accept: */*'
```

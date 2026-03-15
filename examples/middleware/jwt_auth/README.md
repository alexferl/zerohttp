# JWT Authentication Example

This example demonstrates zerohttp's JWT authentication middleware using HS256.

## Features

- HS256 JWT token signing
- Access token generation with configurable TTL
- Required claims validation
- Exempt paths for public endpoints
- Scope-based access control

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint           | Auth Required | Description                           |
|--------------------|---------------|---------------------------------------|
| `POST /login`      | No            | Authenticate and get JWT token        |
| `POST /register`   | No            | Registration stub                     |
| `GET /api/profile` | Yes           | Get user profile                      |
| `GET /api/admin`   | Yes + scope   | Admin endpoint (requires admin scope) |

## Credentials

- Username: `alice`
- Password: `secret`

## Test Commands

### Login and get token
```bash
curl -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret"}'
```

### Access protected endpoint
```bash
curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/profile
```

### Access admin endpoint (fails without admin scope)
```bash
curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/admin
```

### Access without token (fails)
```bash
curl http://localhost:8080/api/profile
```

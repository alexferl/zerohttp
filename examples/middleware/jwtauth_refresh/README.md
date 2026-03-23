# JWT Auth with Refresh Example

This example demonstrates JWT authentication with refresh token support using zerohttp.

## Features

- JWT token generation with HS256
- Access tokens (short-lived) and refresh tokens (long-lived)
- Token revocation support
- Protected routes

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint             | Description                                  |
|----------------------|----------------------------------------------|
| `POST /login`        | Get access and refresh tokens                |
| `POST /logout`       | Revoke refresh token                         |
| `POST /register`     | Register a new user (stub)                   |
| `POST /auth/refresh` | Refresh tokens (checks revocation)           |
| `GET /api/profile`   | Get user profile (requires auth)             |
| `GET /api/admin`     | Admin endpoint (requires auth + admin scope) |

## Test Commands

### Login and get tokens
```bash
curl -X POST http://localhost:8080/login -d '{"username":"alice","password":"secret"}'
```

### Access protected endpoint
```bash
curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/profile
```

### Refresh tokens (will fail if revoked)
```bash
curl -X POST http://localhost:8080/auth/refresh -d '{"refresh_token":"<refresh_token>"}'
```

### Logout (revokes refresh token)
```bash
curl -X POST http://localhost:8080/logout -d '{"refresh_token":"<refresh_token>"}'
```

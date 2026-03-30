# JWT Authentication with Cookies Example

This example demonstrates zerohttp's JWT authentication middleware using cookies for refresh tokens.

## Features

- HS256 JWT token signing
- Access tokens (short-lived) + Refresh tokens (long-lived)
- Cookie-based refresh token storage (HttpOnly, Secure, SameSite)
- Token revocation support
- Automatic token extraction from cookies or headers
- Required claims validation
- Exclude paths for public endpoints

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

### Web Demo

Open `http://localhost:8080` in your browser to see an interactive demo showing:
- HttpOnly cookies are invisible to JavaScript (protected from XSS)
- The browser automatically sends cookies with API requests
- Token refresh and logout flows

## Endpoints

| Endpoint             | Auth Required | Description                           |
|----------------------|---------------|---------------------------------------|
| `POST /login`        | No            | Authenticate and get tokens           |
| `POST /register`     | No            | Registration stub                     |
| `POST /auth/refresh` | No            | Refresh access token (uses cookie)    |
| `POST /auth/logout`  | No            | Revoke refresh token and clear cookie |
| `GET /api/profile`   | Yes           | Get user profile                      |
| `GET /api/admin`     | Yes + scope   | Admin endpoint (requires admin scope) |

## Credentials

- Username: `alice`
- Password: `secret`

## Token Flow

1. **Login**: Returns access token in JSON, sets refresh token as HttpOnly cookie
2. **Access API**: Send access token in `Authorization: Bearer <token>` header
3. **Refresh**: Hit `/auth/refresh` with the refresh token cookie to get new access token
4. **Logout**: Hit `/auth/logout` to revoke the refresh token and clear the cookie

## Test Commands

### Login (stores refresh token in cookie file)
```bash
curl -c cookies.txt -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"secret"}'
```

### Access protected endpoint with access token header
```bash
curl -H 'Authorization: Bearer <access_token>' http://localhost:8080/api/profile
```

### Refresh access token (uses cookie)
```bash
curl -c cookies.txt -b cookies.txt -X POST http://localhost:8080/auth/refresh
```

### Logout (revokes token and clears cookie)
```bash
curl -b cookies.txt -X POST http://localhost:8080/auth/logout
```

### Access without token (fails)
```bash
curl http://localhost:8080/api/profile
```

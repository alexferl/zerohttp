# JWT Authentication with golang-jwt/jwt Example

This example demonstrates zerohttp's JWT authentication middleware using the popular `github.com/golang-jwt/jwt` library.

## Features

- Custom TokenStore implementation using golang-jwt/jwt v5
- Access and refresh token support
- Token rotation with single-use refresh tokens
- Session-based revocation (refresh revokes both tokens)
- Protected and public endpoints

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint              | Auth Required | Description                 |
|-----------------------|---------------|-----------------------------|
| `POST /login`         | No            | Authenticate and get tokens |
| `POST /auth/refresh`  | No            | Refresh access token        |
| `GET /api/profile`    | Yes           | Get user profile            |

## Credentials

- Username: `alice`
- Password: `secret`

## Test Commands

### Login and extract tokens with jq
```bash
TOKENS=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret"}')
ACCESS_TOKEN=$(echo $TOKENS | jq -r '.access_token')
REFRESH_TOKEN=$(echo $TOKENS | jq -r '.refresh_token')
```

### Access protected endpoint
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/profile
```

### Refresh tokens (revokes old session)
```bash
NEW_TOKENS=$(curl -s -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
NEW_ACCESS_TOKEN=$(echo $NEW_TOKENS | jq -r '.access_token')
NEW_REFRESH_TOKEN=$(echo $NEW_TOKENS | jq -r '.refresh_token')
```

### Try old token after refresh (fails - session revoked)
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/profile
```

### Access without token (fails)
```bash
curl http://localhost:8080/api/profile
```

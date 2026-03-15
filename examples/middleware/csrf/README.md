# CSRF Protection Example

This example demonstrates zerohttp's CSRF middleware using the double-submit cookie pattern.

## Features

- CSRF token generation and validation
- Form token lookup for traditional HTML forms
- Cookie-based token storage (HttpOnly, Secure, SameSite)
- AJAX/Fetch API support

## Running the Example

```bash
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint           | Description                    |
|--------------------|--------------------------------|
| `GET /`            | Overview and documentation     |
| `GET /form`        | HTML form with CSRF protection |
| `POST /submit`     | Form submission handler        |
| `GET /api`         | AJAX/Fetch API demo            |
| `POST /api/update` | API endpoint protected by CSRF |

## Test Commands

### Without CSRF token (will fail with 403)
```bash
curl -X POST http://localhost:8080/submit -d "message=hello"
```

### Get CSRF token (save cookie jar)
```bash
curl -s http://localhost:8080/form -c cookies.txt > /dev/null
```

### Submit with token from cookie jar
```bash
CSRF_TOKEN=$(grep csrf_token cookies.txt | tail -1 | awk '{print $7}')
curl -X POST http://localhost:8080/submit -b cookies.txt -d "csrf_token=$CSRF_TOKEN" -d "message=hello"
```

# SCS Session Management Example

This example demonstrates session management using [SCS](https://github.com/alexedwards/scs) with zerohttp.

SCS provides secure cookie-based session management with configurable storage backends.

## Features

- Cookie-based session tokens
- Session middleware integration
- Protected routes with auth middleware
- Session data persistence (username, login time)

## Running the Example

```bash
# Install dependencies
go mod tidy

# Run the server
go run .
```

The server starts on `http://localhost:8080`.

## Endpoints

| Endpoint       | Description                             |
|----------------|-----------------------------------------|
| `GET /`        | Home page with login link               |
| `GET /login`   | Login form                              |
| `POST /login`  | Submit credentials (admin/password)     |
| `GET /profile` | Protected profile page (requires login) |
| `GET /logout`  | Destroy session and logout              |

## Demo

1. Visit `http://localhost:8080` - you'll see the home page
2. Click "Login" and enter:
   - Username: `admin`
   - Password: `password`
3. You'll be redirected to `/profile` showing your session info
4. Click "Logout" to end the session

Trying to access `/profile` without logging in redirects to `/login`.
